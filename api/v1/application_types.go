/*
Copyright 2024 Aloys.Zhou.

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

package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type DeploymentTemplate struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// omitempty 意味着在编码（序列化）结构体为 JSON 字符串时，如果该字段的值是其零值（zero value），则该字段将不会出现在生成的 JSON 字符串中
	appsv1.DeploymentSpec `json:",omitempty"`
}

type ServiceTemplate struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	corev1.ServiceSpec `json:",omitempty"`
}

// ApplicationSpec defines the desired state of Application.
// 自定义资源的字段，就是cr yaml里面要填写的信息
type ApplicationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Application. Edit application_types.go to remove/update
	// Foo string `json:"foo,omitempty"`
	Deployment DeploymentTemplate `json:"deployment,omitempty"`
	Service    ServiceTemplate    `json:"service,omitempty"`
}

// ApplicationStatus defines the observed state of Application.
// 并不是严格对应的“实际状态”，而是观察记录下的当前对象的最新“状态”
type ApplicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Workflow appsv1.DeploymentStatus `json:"workflow,omitempty"`
	Network  corev1.ServiceStatus    `json:"network,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.deployment.replicas"
// +kubebuilder:printcolumn:name="UpdatedReplicas",type="string",JSONPath=".spec.deployment.replicas.updatedReplicas"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:path=applications,singular=application,scope=Namespaced,shortName=app

// Application is the Schema for the applications API.
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application.
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
