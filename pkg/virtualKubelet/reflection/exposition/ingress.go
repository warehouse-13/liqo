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

package exposition

import (
	"context"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	netv1clients "k8s.io/client-go/kubernetes/typed/networking/v1"
	netv1listers "k8s.io/client-go/listers/networking/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/trace"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/liqotech/liqo/pkg/virtualKubelet/forge"
	"github.com/liqotech/liqo/pkg/virtualKubelet/reflection/generic"
	"github.com/liqotech/liqo/pkg/virtualKubelet/reflection/manager"
	"github.com/liqotech/liqo/pkg/virtualKubelet/reflection/options"
)

var _ manager.NamespacedReflector = (*NamespacedIngressReflector)(nil)

const (
	// IngressReflectorName -> The name associated with the Ingress reflector.
	IngressReflectorName = "Ingress"
)

// NamespacedIngressReflector manages the Ingress reflection for a given pair of local and remote namespaces.
type NamespacedIngressReflector struct {
	generic.NamespacedReflector

	localIngresses        netv1listers.IngressNamespaceLister
	remoteIngresses       netv1listers.IngressNamespaceLister
	remoteIngressesClient netv1clients.IngressInterface
}

// NewIngressReflector returns a new IngressReflector instance.
func NewIngressReflector(workers uint) manager.Reflector {
	return generic.NewReflector(IngressReflectorName, NewNamespacedIngressReflector, generic.WithoutFallback(), workers)
}

// NewNamespacedIngressReflector returns a new NamespacedIngressReflector instance.
func NewNamespacedIngressReflector(opts *options.NamespacedOpts) manager.NamespacedReflector {
	local := opts.LocalFactory.Networking().V1().Ingresses()
	remote := opts.RemoteFactory.Networking().V1().Ingresses()

	local.Informer().AddEventHandler(opts.HandlerFactory(generic.NamespacedKeyer(opts.LocalNamespace)))
	remote.Informer().AddEventHandler(opts.HandlerFactory(generic.NamespacedKeyer(opts.LocalNamespace)))

	return &NamespacedIngressReflector{
		NamespacedReflector:   generic.NewNamespacedReflector(opts),
		localIngresses:        local.Lister().Ingresses(opts.LocalNamespace),
		remoteIngresses:       remote.Lister().Ingresses(opts.RemoteNamespace),
		remoteIngressesClient: opts.RemoteClient.NetworkingV1().Ingresses(opts.RemoteNamespace),
	}
}

// Handle reconciles ingress objects.
func (nir *NamespacedIngressReflector) Handle(ctx context.Context, name string) error {
	tracer := trace.FromContext(ctx)

	// Retrieve the local and remote objects (only not found errors can occur).
	klog.V(4).Infof("Handling reflection of local Ingress %q (remote: %q)", nir.LocalRef(name), nir.RemoteRef(name))
	local, lerr := nir.localIngresses.Get(name)
	utilruntime.Must(client.IgnoreNotFound(lerr))
	remote, rerr := nir.remoteIngresses.Get(name)
	utilruntime.Must(client.IgnoreNotFound(rerr))
	tracer.Step("Retrieved the local and remote objects")

	// Abort the reflection if the remote object is not managed by us, as we do not want to mutate others' objects.
	if rerr == nil && !forge.IsReflected(remote) {
		klog.Infof("Skipping reflection of local Ingress %q as remote already exists and is not managed by us", nir.LocalRef(name))
		return nil
	}
	tracer.Step("Performed the sanity checks")

	// The local ingress does no longer exist. Ensure it is also absent from the remote cluster.
	if kerrors.IsNotFound(lerr) {
		defer tracer.Step("Ensured the absence of the remote object")
		if !kerrors.IsNotFound(rerr) {
			klog.V(4).Infof("Deleting remote Ingress %q, since local %q does no longer exist", nir.RemoteRef(name), nir.LocalRef(name))
			return nir.DeleteRemote(ctx, nir.remoteIngressesClient, IngressReflectorName, name, remote.GetUID())
		}

		klog.V(4).Infof("Local Ingress %q and remote Ingress %q both vanished", nir.LocalRef(name), nir.RemoteRef(name))
		return nil
	}

	// Forge the mutation to be applied to the remote cluster.
	mutation := forge.RemoteIngress(local, nir.RemoteNamespace())
	tracer.Step("Remote mutation created")

	defer tracer.Step("Enforced the correctness of the remote object")
	if _, err := nir.remoteIngressesClient.Apply(ctx, mutation, forge.ApplyOptions()); err != nil {
		klog.Errorf("Failed to enforce remote Ingress %q (local: %q): %v", nir.RemoteRef(name), nir.LocalRef(name), err)
		return err
	}

	klog.Infof("Remote Ingress %q successfully enforced (local: %q)", nir.RemoteRef(name), nir.LocalRef(name))
	return nil
}
