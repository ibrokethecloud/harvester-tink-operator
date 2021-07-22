/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	installer "github.com/ibrokethecloud/harvester-tink-operator/pkg/installer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	Token             string            `json:"token"`
	Nameservers       []string          `json:"nameServers,omitempty"`
	Netmask           string            `json:"netmask,omitempty"`
	Gateway           string            `json:"gateway,omitempty"`
	PXEIsoURL         string            `json:"pxeIsoURL,omitempty"`
	SSHAuthorizedKeys []string          `json:"sshAuthorizedKeys,omitempty"`
	Modules           []string          `json:"modules,omitempty"`
	Sysctls           map[string]string `json:"sysctls,omitempty"`
	NTPServers        []string          `json:"ntpServers,omitempty"`
	DNSNameservers    []string          `json:"dnsNameservers,omitempty"`
	Wifi              []installer.Wifi  `json:"wifi,omitempty"`
	Password          string            `json:"password,omitempty"`
	Environment       map[string]string `json:"environment,omitempty"`
	Disk              string            `json:"disk,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	NodeStatus map[string]string
	Status     string
	Message    string
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope="Cluster"
// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
