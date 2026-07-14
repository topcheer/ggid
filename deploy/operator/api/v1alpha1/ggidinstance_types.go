package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GGIDInstanceSpec defines the desired state of GGIDInstance.
// A GGIDInstance represents a dedicated IAM deployment for a large customer.
type GGIDInstanceSpec struct {
	// Tier is always "dedicated" for GGIDInstance.
	// +kubebuilder:default:=dedicated
	Tier string `json:"tier"`

	// OrganizationName is the display name of the organization.
	OrganizationName string `json:"organizationName"`

	// Namespace is the Kubernetes namespace to deploy into.
	// If empty, defaults to "ggid-<instance-name>".
	Namespace string `json:"namespace,omitempty"`

	// Replicas is the number of replicas for each microservice.
	// +kubebuilder:default:=2
	Replicas int32 `json:"replicas,omitempty"`

	// Database is the database configuration.
	Database DatabaseConfig `json:"database"`

	// AdminEmail is the initial admin user's email.
	AdminEmail string `json:"adminEmail"`

	// IdPConfig is optional external IdP configuration.
	// +optional
	IdPConfig *IdPConfig `json:"idpConfig,omitempty"`

	// HelmChartRef is the Helm chart reference to use.
	// If empty, uses the default GGID Helm chart.
	// +optional
	HelmChartRef string `json:"helmChartRef,omitempty"`
}

// DatabaseConfig defines database connection parameters.
type DatabaseConfig struct {
	// Driver: postgres, mysql, or sqlite. Default: postgres.
	// +kubebuilder:default:=postgres
	Driver string `json:"driver"`

	// Host is the database host.
	Host string `json:"host"`

	// Port is the database port.
	// +kubebuilder:default:=5432
	Port int32 `json:"port,omitempty"`

	// Name is the database name.
	// +kubebuilder:default:=ggid
	Name string `json:"name,omitempty"`

	// SSLMode for PostgreSQL (disable, require, verify-ca, verify-full).
	// +kubebuilder:default:=require
	SSLMode string `json:"sslMode,omitempty"`
}

// IdPConfig defines external Identity Provider configuration.
type IdPConfig struct {
	// Provider type: saml, oidc, or ldap.
	Provider string `json:"provider"`

	// EntityID or Issuer URL.
	EntityID string `json:"entityId"`

	// SSOURL is the IdP single sign-on endpoint.
	SSOURL string `json:"ssoUrl"`

	// Certificate is the IdP's X.509 certificate (PEM, base64-encoded in YAML).
	Certificate string `json:"certificate,omitempty"`
}

// GGIDInstanceStatus defines the observed state of GGIDInstance.
type GGIDInstanceStatus struct {
	// Phase: Pending, Provisioning, Ready, Failed, Deleting.
	Phase string `json:"phase,omitempty"`

	// Namespace is the namespace where the instance was deployed.
	Namespace string `json:"namespace,omitempty"`

	// HelmRelease is the name of the Helm release.
	HelmRelease string `json:"helmRelease,omitempty"`

	// TenantID is the UUID assigned to this instance's default tenant.
	TenantID string `json:"tenantId,omitempty"`

	// Conditions represents the latest available observations.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ggi

// GGIDInstance is the Schema for the ggidinstances API.
// It represents a dedicated GGID IAM deployment.
type GGIDInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GGIDInstanceSpec   `json:"spec,omitempty"`
	Status GGIDInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GGIDInstanceList contains a list of GGIDInstance.
type GGIDInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GGIDInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GGIDInstance{}, &GGIDInstanceList{})
}
