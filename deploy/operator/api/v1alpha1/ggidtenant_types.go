package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GGIDTenantSpec defines the desired state of GGIDTenant.
// A GGIDTenant represents a tenant within a shared GGID instance.
type GGIDTenantSpec struct {
	// Tier is always "shared" for GGIDTenant.
	// +kubebuilder:default:=shared
	Tier string `json:"tier"`

	// TenantName is the display name.
	TenantName string `json:"tenantName"`

	// Slug is the URL-friendly identifier. Must be unique.
	Slug string `json:"slug"`

	// Plan: free, starter, pro, enterprise.
	// +kubebuilder:default:=starter
	Plan string `json:"plan,omitempty"`

	// MaxUsers is the maximum number of users allowed.
	// +kubebuilder:default:=1000
	MaxUsers int32 `json:"maxUsers,omitempty"`

	// AdminEmail is the initial admin user's email.
	AdminEmail string `json:"adminEmail"`

	// AdminPassword is the initial admin's password.
	// This should be referenced from a Secret in production.
	AdminPassword string `json:"adminPassword,omitempty"`

	// GGIDInstanceRef is the name of the GGIDInstance this tenant belongs to.
	// If empty, uses the default shared instance.
	// +optional
	GGIDInstanceRef string `json:"ggidInstanceRef,omitempty"`

	// IdPConfig is optional external IdP configuration for this tenant.
	// +optional
	IdPConfig *IdPConfig `json:"idpConfig,omitempty"`
}

// GGIDTenantStatus defines the observed state of GGIDTenant.
type GGIDTenantStatus struct {
	// Phase: Pending, Provisioning, Ready, Failed, Suspending, Suspended.
	Phase string `json:"phase,omitempty"`

	// TenantID is the UUID assigned by the GGID Org service.
	TenantID string `json:"tenantId,omitempty"`

	// GatewayURL is the API gateway URL for this tenant.
	GatewayURL string `json:"gatewayUrl,omitempty"`

	// Conditions represents the latest available observations.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ggt

// GGIDTenant is the Schema for the ggidtenants API.
// It represents a tenant within a shared GGID instance.
type GGIDTenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GGIDTenantSpec   `json:"spec,omitempty"`
	Status GGIDTenantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GGIDTenantList contains a list of GGIDTenant.
type GGIDTenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GGIDTenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GGIDTenant{}, &GGIDTenantList{})
}
