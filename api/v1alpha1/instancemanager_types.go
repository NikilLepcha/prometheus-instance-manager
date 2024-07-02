package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstanceManagerSpec defines the desired state of InstanceManager
type InstanceManagerSpec struct {
	Name           string `json:"name,omitempty"`
	Replicas       int32  `json:"replicas,omitempty"`
	VolumeSize     int    `json:"volumeSize,omitempty"`
	Image          string `json:"image,omitempty"`
	ScrapeInterval string `json:"scrapeInterval,omitempty"`
}

// InstanceManagerStatus defines the observed state of InstanceManager
type InstanceManagerStatus struct {
	// Name     string `json:"name,omitempty"`
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InstanceManager is the Schema for the instancemanagers API
type InstanceManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceManagerSpec   `json:"spec,omitempty"`
	Status InstanceManagerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InstanceManagerList contains a list of InstanceManager
type InstanceManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstanceManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstanceManager{}, &InstanceManagerList{})
}
