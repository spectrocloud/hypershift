package v1beta1

// MAASPlatformSpec specifies configuration for clusters running on MaaS (Metal as a Service).
type MAASPlatformSpec struct {
	// identityRef is a reference to a secret holding MAAS credentials
	// to be used when reconciling the hosted cluster.
	//
	// +kubebuilder:validation:Required
	// +required
	IdentityRef MAASIdentityReference `json:"identityRef"`

	// dnsDomain is the DNS domain for the MAAS cluster.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	DNSDomain string `json:"dnsDomain,omitempty"`

	// zone specifies the MAAS zone where the cluster will be deployed.
	// If not specified, the cluster will be deployed in any available zone.
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Zone string `json:"zone,omitempty"`
}

// MAASIdentityReference is a reference to an infrastructure
// provider identity to be used to provision cluster resources.
type MAASIdentityReference struct {
	// Name is the name of a secret in the same namespace as the resource being provisioned.
	// The secret must contain the following keys:
	// - `MAAS_ENDPOINT`: MAAS API endpoint URL
	// - `MAAS_API_KEY`: MAAS API key for authentication
	//
	// +kubebuilder:validation:Required
	// +required
	Name string `json:"name"`
}

// MAASNodePoolPlatform specifies the configuration for MaaS platform.
type MAASNodePoolPlatform struct {
	// identityRef is a reference to a secret holding MAAS credentials
	// to be used when reconciling the node pool.
	// The secret must contain the following keys:
	// - `MAAS_ENDPOINT`: MAAS API endpoint URL
	// - `MAAS_API_KEY`: MAAS API key for authentication
	//
	// +kubebuilder:validation:Required
	// +required
	IdentityRef MAASIdentityReference `json:"identityRef"`

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

	// minDiskSize specifies the minimum disk size in GB required for the nodes.
	// +optional
	// +kubebuilder:validation:Minimum=1
	MinDiskSize *int32 `json:"minDiskSize,omitempty"`

	// lxd contains configuration for creating this machine as an LXD VM on a host
	// when enabled. When nil or disabled, this machine is created on bare metal.
	// +optional
	LXD *MAASLXDConfig `json:"lxd,omitempty"`

	// staticIP configuration for VMs
	// +optional
	StaticIP *MAASStaticIPConfig `json:"staticIP,omitempty"`
}

// MAASLXDConfig defines LXD VM creation options for a machine
type MAASLXDConfig struct {
	// enabled specifies whether this machine should be created as an LXD VM
	// +kubebuilder:default=false
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// storagePool is the storage pool to use for the VM
	// +optional
	// +kubebuilder:validation:MaxLength=255
	StoragePool string `json:"storagePool,omitempty"`

	// network is the network to connect the VM to
	// +optional
	// +kubebuilder:validation:MaxLength=255
	Network string `json:"network,omitempty"`
}

// MAASStaticIPConfig defines the static IP configuration for a VM
type MAASStaticIPConfig struct {
	// ip is the static IP address to assign
	// +optional
	IP string `json:"ip,omitempty"`

	// cidr is the network CIDR
	// +optional
	CIDR string `json:"cidr,omitempty"`

	// gateway is the network gateway
	// +optional
	Gateway string `json:"gateway,omitempty"`

	// nameservers is a list of DNS servers
	// +optional
	Nameservers []string `json:"nameservers,omitempty"`
}
