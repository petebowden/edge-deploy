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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EdgeDeploymentSpec defines the desired state of EdgeDeployment
type EdgeDeploymentSpec struct {
	// Edge Containers
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

// EdgeDeploymentStatus defines the observed state of EdgeDeployment
type EdgeDeploymentStatus struct {
	// +kubebuilder:validation:XEmbeddedResource
	// Using Podspec for now. Will eventually need to create another CRD
	Spec corev1.PodSpec `json:"podspec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EdgeDeployment is the Schema for the edgedeployments API
// +operator-sdk:csv:customresourcedefinitions:displayName="Edge Deployment",resources={{PodSpec,v1,deployment-spec}}
type EdgeDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeDeploymentSpec   `json:"spec,omitempty"`
	Status EdgeDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EdgeDeploymentList contains a list of EdgeDeployment
type EdgeDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeDeployment{}, &EdgeDeploymentList{})
}
