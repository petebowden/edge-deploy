/*
Copyright 2021.

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

// EdgePodSpec defines the desired state of EdgePod
type EdgePodSpec struct {
	Containers []EdgeContainer `json:"containers"`
}

// EdgeContainer Object
type EdgeContainer struct {
	Name  string          `json:"name"`
	Image string          `json:"image"`
	Ports []ContainerPort `json:"ports"`
}

// ContainerPort holds port information
type ContainerPort struct {
	// +optional
	HostPort      int32 `json:"hostPort"`
	ContainerPort int32 `json:"containerPort"`
}

// EdgePodStatus defines the observed state of EdgePod
type EdgePodStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EdgePod is the Schema for the edgepods API
type EdgePod struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Is this valid?
	EdgeTarget string           `json:"edgetarget,omitempty"`
	Podspec    *InternalPodspec `json:"podspec,omitempty"`

	Spec   EdgePodSpec   `json:"spec,omitempty"`
	Status EdgePodStatus `json:"status,omitempty"`
}

type InternalPodspec struct {
	ApiVersion        string `json:"apiversion,omitempty"`
	Kind              string `json:"kind,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EdgePodSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// EdgePodList contains a list of EdgePod
type EdgePodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgePod `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgePod{}, &EdgePodList{})
}
