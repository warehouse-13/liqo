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

package consts

const (
	// ClusterNameParameter is the name of the parameter specifying the cluster name.
	ClusterNameParameter = "cluster-name"
	// ClusterLabelsParameter is the name of the parameter specifying the cluster labels.
	ClusterLabelsParameter = "cluster-labels"
	// ReservedSubnetsParameter is the name of the parameter specifying the cluster's reserved subnets.
	ReservedSubnetsParameter = "reserved-subnets"
	// EnableLanDiscoveryParameter is the name of the parameter specifying whether the lan discovery is enabled.
	EnableLanDiscoveryParameter = "enable-lan-discovery"
	// GenerateNameParameter is the name of the parameter specifying whether to generate a random name for the cluster.
	GenerateNameParameter = "generate-name"

	// AuthServiceAddressOverrideParameter is the name of the parameter overriding
	// the automatically detected authentication service address.
	AuthServiceAddressOverrideParameter = "auth-service-address-override"
	// AuthServicePortOverrideParameter is the name of the parameter overriding
	// the automatically detected authentication service address.
	AuthServicePortOverrideParameter = "auth-service-port-override"

	// ExternalResourceMonitorParameter is the name of the parameter specifying the address of an ExternalResourceMonitor.
	ExternalResourceMonitorParameter = "external-monitor"
)
