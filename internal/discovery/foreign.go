package discovery

import (
	"context"
	"errors"
	"github.com/liqotech/liqo/apis/discovery/v1alpha1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog"
	"strings"
)

// 1. checks if cluster ID is already known
// 2. if not exists, create it
// 3. else
//   3a. if IP is different set new IP and delete CA data
//   3b. else it is ok
// 4. TTL logic

func (discovery *DiscoveryCtrl) UpdateForeign(data []*TxtData, sd *v1alpha1.SearchDomain) []*v1alpha1.ForeignCluster {
	createdUpdatedForeign := []*v1alpha1.ForeignCluster{}
	var discoveryType v1alpha1.DiscoveryType
	if sd == nil {
		discoveryType = v1alpha1.LanDiscovery
	} else {
		discoveryType = v1alpha1.WanDiscovery
	}
	for _, txtData := range data {
		if txtData.ID == discovery.ClusterId.GetClusterID() {
			// is local cluster
			continue
		}
		fc, err := discovery.GetForeignClusterByID(txtData.ID)
		if k8serror.IsNotFound(err) {
			fc, err := discovery.createForeign(txtData, sd, discoveryType)
			if err != nil {
				klog.Error(err, err.Error())
				continue
			}
			klog.Info("ForeignCluster " + txtData.ID + " created")
			createdUpdatedForeign = append(createdUpdatedForeign, fc)
		} else if err == nil {
			fc, err = discovery.CheckUpdate(txtData, fc, discoveryType, sd)
			if err != nil {
				klog.Error(err, err.Error())
				continue
			}
			klog.Info("ForeignCluster " + txtData.ID + " updated")
			createdUpdatedForeign = append(createdUpdatedForeign, fc)
		} else {
			// unhandled errors
			klog.Error(err, err.Error())
			continue
		}
	}
	if discoveryType == v1alpha1.LanDiscovery {
		_ = discovery.UpdateTtl(data)
	}
	return createdUpdatedForeign
}

// this function is called every x seconds when LAN discovery is triggered
// for each cluster with discovery-type = LAN we will decrease TTL if that cluster
// didn't answered to current discovery
// when TTL is 0 that ForeignCluster will be deleted
func (discovery *DiscoveryCtrl) UpdateTtl(txts []*TxtData) error {
	// find all ForeignCluster with discovery type LAN
	tmp, err := discovery.crdClient.Resource("foreignclusters").List(metav1.ListOptions{
		LabelSelector: "discovery-type=LAN",
	})
	if err != nil {
		klog.Error(err, err.Error())
		return err
	}
	fcs, ok := tmp.(*v1alpha1.ForeignClusterList)
	if !ok {
		err = errors.New("retrieved object is not a ForeignClusterList")
		klog.Error(err, err.Error())
		return err
	}
	for i := range fcs.Items {
		fc := fcs.Items[i]
		// find the ones that are not in the last retrieved list on LAN
		found := false
		for _, txt := range txts {
			if txt.ID == fc.Spec.ClusterIdentity.ClusterID {
				found = true
				// if cluster TTL was decreased, reset it to default value
				if fc.Status.Ttl != 3 {
					fc.Status.Ttl = 3
					_, err = discovery.crdClient.Resource("foreignclusters").Update(fc.Name, &fc, metav1.UpdateOptions{})
					if err != nil {
						klog.Error(err, err.Error())
						continue
					}
				}
				break
			}
		}
		if !found {
			// if ForeignCluster is not in Txt list, reduce its TTL
			fc.Status.Ttl -= 1
			if fc.Status.Ttl <= 0 {
				// delete ForeignCluster
				err = discovery.crdClient.Resource("foreignclusters").Delete(fc.Name, metav1.DeleteOptions{})
				if err != nil {
					klog.Error(err, err.Error())
					continue
				}
			} else {
				// update ForeignCluster
				_, err = discovery.crdClient.Resource("foreignclusters").Update(fc.Name, &fc, metav1.UpdateOptions{})
				if err != nil {
					klog.Error(err, err.Error())
					continue
				}
			}
		}
	}
	return nil
}

func (discovery *DiscoveryCtrl) createForeign(txtData *TxtData, sd *v1alpha1.SearchDomain, discoveryType v1alpha1.DiscoveryType) (*v1alpha1.ForeignCluster, error) {
	fc := &v1alpha1.ForeignCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: txtData.ID,
		},
		Spec: v1alpha1.ForeignClusterSpec{
			ClusterIdentity: v1alpha1.ClusterIdentity{
				ClusterID:   txtData.ID,
				ClusterName: txtData.Name,
			},
			Namespace:     txtData.Namespace,
			ApiUrl:        txtData.ApiUrl,
			DiscoveryType: discoveryType,
		},
	}

	if sd != nil {
		fc.Spec.Join = sd.Spec.AutoJoin
		fc.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: "discovery.liqo.io/v1alpha1",
				Kind:       "SearchDomain",
				Name:       sd.Name,
				UID:        sd.UID,
			},
		}
	}
	if discoveryType == v1alpha1.LanDiscovery {
		// set TTL to default value
		fc.Status.Ttl = 3
	}
	tmp, err := discovery.crdClient.Resource("foreignclusters").Create(fc, metav1.CreateOptions{})
	if err != nil {
		klog.Error(err, err.Error())
		return nil, err
	}
	fc, ok := tmp.(*v1alpha1.ForeignCluster)
	if !ok {
		return nil, errors.New("created object is not a ForeignCluster")
	}
	return fc, err
}

func (discovery *DiscoveryCtrl) CheckUpdate(txtData *TxtData, fc *v1alpha1.ForeignCluster, discoveryType v1alpha1.DiscoveryType, searchDomain *v1alpha1.SearchDomain) (*v1alpha1.ForeignCluster, error) {
	if fc.Spec.ApiUrl != txtData.ApiUrl || fc.Spec.Namespace != txtData.Namespace || fc.HasHigherPriority(discoveryType) {
		fc.Spec.ApiUrl = txtData.ApiUrl
		fc.Spec.Namespace = txtData.Namespace
		fc.Spec.DiscoveryType = discoveryType
		if searchDomain != nil && discoveryType == v1alpha1.WanDiscovery {
			fc.Spec.Join = searchDomain.Spec.AutoJoin
		}
		if fc.Status.Outgoing.CaDataRef != nil {
			err := discovery.crdClient.Client().CoreV1().Secrets(fc.Status.Outgoing.CaDataRef.Namespace).Delete(context.TODO(), fc.Status.Outgoing.CaDataRef.Name, metav1.DeleteOptions{})
			if err != nil {
				klog.Error(err, err.Error())
				return nil, err
			}
		}
		fc.Status.Outgoing.CaDataRef = nil
		tmp, err := discovery.crdClient.Resource("foreignclusters").Update(fc.Name, fc, metav1.UpdateOptions{})
		if err != nil {
			klog.Error(err, err.Error())
			return nil, err
		}
		fc, ok := tmp.(*v1alpha1.ForeignCluster)
		if !ok {
			err = errors.New("retrieved object is not a ForeignCluster")
			klog.Error(err, err.Error())
			return nil, err
		}
		if fc.Status.Outgoing.Advertisement != nil {
			// changed ip in peered cluster, delete advertisement and wait for its recreation
			// TODO: find more sophisticated logic to not remove all resources on remote cluster
			advName := fc.Status.Outgoing.Advertisement.Name
			fc.Status.Outgoing.Advertisement = nil
			// updating it before adv delete will avoid us to set to false join flag
			tmp, err = discovery.crdClient.Resource("foreignclusters").Update(fc.Name, fc, metav1.UpdateOptions{})
			if err != nil {
				klog.Error(err, err.Error())
				return nil, err
			}
			fc, ok = tmp.(*v1alpha1.ForeignCluster)
			if !ok {
				err = errors.New("retrieved object is not a ForeignCluster")
				klog.Error(err, err.Error())
				return nil, err
			}
			err = discovery.advClient.Resource("advertisements").Delete(advName, metav1.DeleteOptions{})
			if err != nil {
				klog.Error(err, err.Error())
				return nil, err
			}
		}
		return fc, nil
	}
	return fc, nil
}

func (discovery *DiscoveryCtrl) GetForeignClusterByID(clusterID string) (*v1alpha1.ForeignCluster, error) {
	tmp, err := discovery.crdClient.Resource("foreignclusters").List(metav1.ListOptions{
		LabelSelector: strings.Join([]string{"cluster-id", clusterID}, "="),
	})
	if err != nil {
		return nil, err
	}
	fcs, ok := tmp.(*v1alpha1.ForeignClusterList)
	if !ok || len(fcs.Items) == 0 {
		return nil, k8serror.NewNotFound(schema.GroupResource{
			Group:    v1alpha1.GroupVersion.Group,
			Resource: "foreignclusters",
		}, clusterID)
	}
	return &fcs.Items[0], nil
}
