// Copyright 2019-2022 The Liqo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubeadm

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/liqotech/liqo/pkg/liqoctl/common"
	logsutils "github.com/liqotech/liqo/pkg/utils/logs"
)

func retrieveClusterParameters(ctx context.Context, client kubernetes.Interface) (podCIDR, serviceCIDR string, err error) {
	kubeControllerSpec, err := client.CoreV1().Pods(kubeSystemNamespaceName).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set(kubeControllerManagerLabels).AsSelector().String(),
	})
	if err != nil {
		return "", "", err
	}
	if len(kubeControllerSpec.Items) < 1 {
		return "", "", fmt.Errorf("kube-controller-manager not found")
	}
	if len(kubeControllerSpec.Items[0].Spec.Containers) != 1 {
		return "", "", fmt.Errorf("unexpected amount of containers in kube-controller-manager")
	}

	command := kubeControllerSpec.Items[0].Spec.Containers[0].Command
	podCIDR = common.ExtractValuesFromArgumentListOrDefault(podCIDRParameterFilter, command, defaultPodCIDR)
	logsutils.Infof("Extracted podCIDR: %s\n", podCIDR)

	serviceCIDR = common.ExtractValuesFromArgumentListOrDefault(serviceCIDRParameterFilter, command, defaultServiceCIDR)
	logsutils.Infof("Extracted serviceCIDR: %s\n", serviceCIDR)

	return podCIDR, serviceCIDR, nil
}
