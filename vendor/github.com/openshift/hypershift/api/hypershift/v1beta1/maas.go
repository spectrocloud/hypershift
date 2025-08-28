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

	// dnsDomain is the DNS domain for the MAAS cluster.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	DNSDomain string `json:"dnsDomain,omitempty"`
}

// MAASNodePoolPlatform specifies the configuration for MaaS platform.
type MAASNodePoolPlatform struct {
	// machineType specifies the type of MAAS machine to use for the nodes.
	// This corresponds to the MAAS machine type/tag that will be used for node selection.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	MachineType string `json:"machineType,omitempty"`

	// zone specifies the MAAS zone where the nodes will be deployed.
	// If not specified, nodes will be deployed in any available zone.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Zone string `json:"zone,omitempty"`

	// tags specifies additional MAAS tags to apply to the nodes for filtering and organization.
	// +optional
	// +kubebuilder:validation:MaxItems=10
	Tags []string `json:"tags,omitempty"`

	// resourcePool specifies the MAAS resource pool to use for node allocation.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	ResourcePool string `json:"resourcePool,omitempty"`

	// minCpu specifies the minimum CPU count required for the nodes.
	// +optional
	// +kubebuilder:validation:Minimum=1
	MinCPU *int32 `json:"minCpu,omitempty"`

	// minMemory specifies the minimum memory in MB required for the nodes.
	// +optional
	// +kubebuilder:validation:Minimum=1024
	MinMemory *int32 `json:"minMemory,omitempty"`

	// image specifies the MAAS image ID to use for the nodes.
	// If not specified, a default image will be used based on the release.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Image string `json:"image,omitempty"`

	// failureDomain specifies the failure domain the machine will be created in.
	// Must match a key in the FailureDomains map stored on the cluster object.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	FailureDomain string `json:"failureDomain,omitempty"`
}
