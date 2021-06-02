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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RegisterSpec defines the desired state of Register
type RegisterSpec struct {
	MacAddress  string   `json:"macAddress"`
	Token       string   `json:"token"`
	Nameservers []string `json:"nameServers,omitempty"`
	Interface   string   `json:"interface"`
	Address     string   `json:"address,omitempty"`
	Netmask     string   `json:"netmask,omitempty"`
	Gateway     string   `json:"gateway,omitempty"`
	IsoURL      string   `json:"isourl,omitempty"`
}

// RegisterStatus defines the observed state of Register
type RegisterStatus struct {
	Message           string `json:"message"`
	Status            string `json:"status"`
	UUID              string `json:"uuid"`
	HardwarePublished bool   `json:"hardwarePublished"`
	NodeReady         bool   `json:"nodeReady"`
}

// +kubebuilder:object:root=true

// Register is the Schema for the registers API
type Register struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegisterSpec   `json:"spec,omitempty"`
	Status RegisterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RegisterList contains a list of Register
type RegisterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Register `json:"items"`
}

type MetaData struct {
	Facility `json:"facility"`
	Instance `json:"instance"`
}

type Facility struct {
	FacilityCode string `json:"facility_code"`
}

type Instance struct {
	UserData        string `json:"userdata"`
	OperatingSystem `json:"operating_system"`
}

type OperatingSystem struct {
	Slug string `json:"slug"`
}

func init() {
	SchemeBuilder.Register(&Register{}, &RegisterList{})
}
