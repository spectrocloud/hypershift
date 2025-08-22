package v1beta1

// MAASPlatformSpec specifies configuration for clusters running on MaaS (Metal as a Service).
type MAASPlatformSpec struct {
	// maasConfig specifies the MaaS configuration for the cluster.
	// +required
	MaaSConfig MaaSConfig `json:"maasConfig"`
}

// MaaSConfig specifies the MaaS API configuration.
type MaaSConfig struct {
	// endpoint is the MaaS API endpoint URL.
	// +required
	// +kubebuilder:validation:MaxLength=255
	Endpoint string `json:"endpoint"`

	// apiKey is the MaaS API key for authentication.
	// +required
	// +kubebuilder:validation:MaxLength=255
	APIKey string `json:"apiKey"`

	// zone is the MaaS zone where the cluster will be deployed.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Zone string `json:"zone,omitempty"`
}
