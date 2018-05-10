package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FloatingIP struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FloatinIPSpec `json:"spec"`
}

// FloatinIPSpec defines a floating ip resource
type FloatinIPSpec struct {
	// Floating IP from Hetzner that will be assigned to nodes matching the
	// nodeSelector
	IP string `json:"IP"`

	// Query to select a pool of nodes that
	NodeSelector map[string]string `json:"nodeSelector"`

	// Frequency for reconcilation loops
	IntervalSeconds Seconds `json:"intervalSeconds,omitempty"`
}

// Seconds is an duration in seconds
type Seconds int64

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FloatingIPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []FloatingIP `json:"items"`
}
