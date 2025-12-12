/*
Copyright 2025.

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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TigerBeetleSpec defines the desired state of TigerBeetle
type TigerBeetleSpec struct {
	// ClusterID is the TigerBeetle cluster identifier
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// +optional
	ClusterID int `json:"clusterId,omitempty"`

	// Replicas is the number of TigerBeetle replicas
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=6
	// +kubebuilder:default=3
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Image configuration for the TigerBeetle container
	// +optional
	Image ImageSpec `json:"image,omitempty"`

	// Storage configuration for TigerBeetle data
	// +optional
	Storage StorageSpec `json:"storage,omitempty"`

	// Resources defines the resource requirements for TigerBeetle pods
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// ImagePullSecrets is a list of references to secrets for pulling images
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// SecurityContext defines the security context for the pods
	// +optional
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// NodeSelector for pod scheduling
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations for pod scheduling
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Affinity rules for pod scheduling
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// PodAntiAffinity configuration
	// +optional
	PodAntiAffinity *PodAntiAffinitySpec `json:"podAntiAffinity,omitempty"`

	// TopologySpreadConstraints for pod scheduling
	// +optional
	TopologySpreadConstraints *TopologySpreadSpec `json:"topologySpreadConstraints,omitempty"`

	// PriorityClassName for the pods
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Service configuration
	// +optional
	Service ServiceSpec `json:"service,omitempty"`

	// Extra environment variables
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Extra command-line arguments
	// +optional
	Args []string `json:"args,omitempty"`

	// Annotations to add to pods
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	// Labels to add to pods
	// +optional
	PodLabels map[string]string `json:"podLabels,omitempty"`
}

// ImageSpec defines the container image configuration
type ImageSpec struct {
	// Repository is the container image repository
	// +kubebuilder:default="ghcr.io/krishnacore/tigerbeetle"
	// +optional
	Repository string `json:"repository,omitempty"`

	// Tag is the container image tag
	// +kubebuilder:default="latest"
	// +optional
	Tag string `json:"tag,omitempty"`

	// PullPolicy defines the image pull policy
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +kubebuilder:default="IfNotPresent"
	// +optional
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
}

// StorageSpec defines the storage configuration
type StorageSpec struct {
	// Size is the size of the persistent volume
	// +kubebuilder:default="10Gi"
	// +optional
	Size resource.Quantity `json:"size,omitempty"`

	// StorageClassName is the name of the StorageClass
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Labels to add to the PVC
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to add to the PVC
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PodAntiAffinitySpec defines pod anti-affinity configuration
type PodAntiAffinitySpec struct {
	// Type of anti-affinity: soft, hard, or empty to disable
	// +kubebuilder:validation:Enum=soft;hard;""
	// +kubebuilder:default="soft"
	// +optional
	Type string `json:"type,omitempty"`

	// TopologyKey for anti-affinity rules
	// +kubebuilder:default="kubernetes.io/hostname"
	// +optional
	TopologyKey string `json:"topologyKey,omitempty"`

	// Weight for soft anti-affinity (1-100)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=100
	// +optional
	Weight int32 `json:"weight,omitempty"`
}

// TopologySpreadSpec defines topology spread constraints
type TopologySpreadSpec struct {
	// MaxSkew describes the degree of pod distribution unevenness
	// +kubebuilder:default=1
	// +optional
	MaxSkew int32 `json:"maxSkew,omitempty"`

	// TopologyKey is the key for the node label
	// +kubebuilder:default="topology.kubernetes.io/zone"
	// +optional
	TopologyKey string `json:"topologyKey,omitempty"`

	// WhenUnsatisfiable indicates how to deal with a pod not satisfying the constraint
	// +kubebuilder:validation:Enum=DoNotSchedule;ScheduleAnyway
	// +kubebuilder:default="ScheduleAnyway"
	// +optional
	WhenUnsatisfiable corev1.UnsatisfiableConstraintAction `json:"whenUnsatisfiable,omitempty"`
}

// ServiceSpec defines the service configuration
type ServiceSpec struct {
	// Type is the Kubernetes service type
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	// +kubebuilder:default="ClusterIP"
	// +optional
	Type corev1.ServiceType `json:"type,omitempty"`

	// Port is the service port
	// +kubebuilder:default=3000
	// +optional
	Port int32 `json:"port,omitempty"`

	// Annotations to add to the service
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// TigerBeetleStatus defines the observed state of TigerBeetle.
type TigerBeetleStatus struct {
	// ReadyReplicas is the number of ready TigerBeetle pods
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Phase represents the current phase of the TigerBeetle cluster
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the current state of the TigerBeetle resource
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicas"
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TigerBeetle is the Schema for the tigerbeetles API
type TigerBeetle struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of TigerBeetle
	// +required
	Spec TigerBeetleSpec `json:"spec"`

	// status defines the observed state of TigerBeetle
	// +optional
	Status TigerBeetleStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// TigerBeetleList contains a list of TigerBeetle
type TigerBeetleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []TigerBeetle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TigerBeetle{}, &TigerBeetleList{})
}
