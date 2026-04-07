package maas

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/cmd/nodepool/core"

	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func DefaultOptions() *RawMAASPlatformCreateOptions {
	return &RawMAASPlatformCreateOptions{
		MAASPlatformOptions: &MAASPlatformOptions{
			LXD:      &MAASLXDConfig{},
			StaticIP: &MAASStaticIPConfig{},
		},
	}
}

type MAASPlatformOptions struct {
	// Basic configuration
	IdentityRef  string
	MachineType  string
	Zone         string
	ResourcePool string
	Tags         []string

	// Resource requirements
	MinCPU    int32
	MinMemory int32
	Image     string

	// Advanced configuration (new fields)
	MinDiskSize *int32
	LXD         *MAASLXDConfig
	StaticIP    *MAASStaticIPConfig
}

type MAASLXDConfig struct {
	Enabled     bool
	StoragePool string
	Network     string
}

type MAASStaticIPConfig struct {
	IP          string
	CIDR        string
	Gateway     string
	Nameservers []string
}

// completedCreateOptions is a private wrapper that enforces a call of Complete() before nodepool creation can be invoked.
type completedMAASPlatformCreateOptions struct {
	*MAASPlatformOptions
}

type MAASPlatformCreateOptions struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedMAASPlatformCreateOptions
}

type RawMAASPlatformCreateOptions struct {
	*MAASPlatformOptions
	// Raw string inputs that need parsing
	TagsRaw        string
	NameserversRaw string
	MinDiskSizeRaw string
	LXDEnabledRaw  string
}

type validatedMAASPlatformCreateOptions struct {
	*completedMAASPlatformCreateOptions
}

type ValidatedMAASPlatformCreateOptions struct {
	*validatedMAASPlatformCreateOptions
}

func (o *ValidatedMAASPlatformCreateOptions) Complete() (*MAASPlatformCreateOptions, error) {
	return &MAASPlatformCreateOptions{
		completedMAASPlatformCreateOptions: &completedMAASPlatformCreateOptions{
			MAASPlatformOptions: o.MAASPlatformOptions,
		},
	}, nil
}

func (o *RawMAASPlatformCreateOptions) Validate() (*ValidatedMAASPlatformCreateOptions, error) {
	// Parse tags
	if o.TagsRaw != "" {
		o.Tags = strings.Split(o.TagsRaw, ",")
		for i, tag := range o.Tags {
			o.Tags[i] = strings.TrimSpace(tag)
		}
	}

	// Parse nameservers
	if o.NameserversRaw != "" {
		o.StaticIP.Nameservers = strings.Split(o.NameserversRaw, ",")
		for i, ns := range o.StaticIP.Nameservers {
			o.StaticIP.Nameservers[i] = strings.TrimSpace(ns)
		}
	}

	// Parse min disk size
	if o.MinDiskSizeRaw != "" {
		size, err := strconv.ParseInt(o.MinDiskSizeRaw, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid min-disk-size: %w", err)
		}
		diskSize := int32(size)
		o.MinDiskSize = &diskSize
	}

	// Parse LXD enabled
	if o.LXDEnabledRaw != "" {
		enabled, err := strconv.ParseBool(o.LXDEnabledRaw)
		if err != nil {
			return nil, fmt.Errorf("invalid lxd-enabled: %w", err)
		}
		o.LXD.Enabled = enabled
	}

	// Validate required fields
	if o.IdentityRef == "" {
		return nil, fmt.Errorf("identity-ref is required")
	}

	// Validate resource requirements
	if o.MinCPU < 1 {
		return nil, fmt.Errorf("min-cpu must be at least 1")
	}
	if o.MinMemory < 1024 {
		return nil, fmt.Errorf("min-memory must be at least 1024 MB")
	}

	return &ValidatedMAASPlatformCreateOptions{
		validatedMAASPlatformCreateOptions: &validatedMAASPlatformCreateOptions{
			completedMAASPlatformCreateOptions: &completedMAASPlatformCreateOptions{
				MAASPlatformOptions: o.MAASPlatformOptions,
			},
		},
	}, nil
}

func (o *MAASPlatformCreateOptions) UpdateNodePool(ctx context.Context, nodePool *hyperv1.NodePool, hcluster *hyperv1.HostedCluster, client crclient.Client) error {
	// Set MAAS platform configuration
	nodePool.Spec.Platform.MAAS = &hyperv1.MAASNodePoolPlatform{
		IdentityRef: hyperv1.MAASIdentityReference{
			Name: o.IdentityRef,
		},
		MachineType:  o.MachineType,
		Zone:         o.Zone,
		ResourcePool: o.ResourcePool,
		Tags:         o.Tags,
		MinCPU:       &o.MinCPU,
		MinMemory:    &o.MinMemory,
		Image:        o.Image,
	}

	// Set advanced configuration if provided
	// TODO: These fields will be available once the API is updated
	// if o.MinDiskSize != nil {
	// 	nodePool.Spec.Platform.MAAS.MinDiskSize = o.MinDiskSize
	// }

	// if o.LXD != nil && o.LXD.Enabled {
	// 	nodePool.Spec.Platform.MAAS.LXD = &hyperv1.MAASLXDConfig{
	// 		Enabled:     &o.LXD.Enabled,
	// 		StoragePool: o.LXD.StoragePool,
	// 		Network:     o.LXD.Network,
	// 	}
	// }

	// if o.StaticIP != nil && o.StaticIP.IP != "" {
	// 	nodePool.Spec.Platform.MAAS.StaticIP = &hyperv1.MAASStaticIPConfig{
	// 		IP:          o.StaticIP.IP,
	// 		CIDR:        o.StaticIP.CIDR,
	// 		Gateway:     o.StaticIP.Gateway,
	// 		Nameservers: o.StaticIP.Nameservers,
	// 	}
	// }

	return nil
}

func (o *MAASPlatformCreateOptions) Type() hyperv1.PlatformType {
	return hyperv1.MAASPlatform
}

func NewCreateCommand(opts *core.CreateNodePoolOptions) *cobra.Command {
	rawOpts := DefaultOptions()
	cmd := &cobra.Command{
		Use:          "maas",
		Short:        "Creates a MAAS NodePool",
		SilenceUsage: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		validatedOpts, err := rawOpts.Validate()
		if err != nil {
			return err
		}

		completedOpts, err := validatedOpts.Complete()
		if err != nil {
			return err
		}

		return opts.CreateNodePool(ctx, completedOpts)
	}

	flags := cmd.Flags()
	rawOpts.AddFlags(flags)

	return cmd
}

func (o *RawMAASPlatformCreateOptions) AddFlags(flags *pflag.FlagSet) {
	// Basic configuration
	flags.StringVar(&o.IdentityRef, "identity-ref", "", "Name of the MAAS credentials secret (required)")
	flags.StringVar(&o.MachineType, "machine-type", "", "MAAS machine type/tag for node selection")
	flags.StringVar(&o.Zone, "zone", "", "MAAS zone where nodes will be deployed")
	flags.StringVar(&o.ResourcePool, "resource-pool", "", "MAAS resource pool for node allocation")
	flags.StringVar(&o.TagsRaw, "tags", "", "Comma-separated list of MAAS tags for filtering")

	// Resource requirements
	flags.Int32Var(&o.MinCPU, "min-cpu", 1, "Minimum CPU count required for nodes")
	flags.Int32Var(&o.MinMemory, "min-memory", 1024, "Minimum memory in MB required for nodes")
	flags.StringVar(&o.Image, "image", "", "MAAS image ID to use for nodes")

	// Advanced configuration
	flags.StringVar(&o.MinDiskSizeRaw, "min-disk-size", "", "Minimum disk size in GB")
	flags.StringVar(&o.LXDEnabledRaw, "lxd-enabled", "false", "Enable LXD VM creation")
	flags.StringVar(&o.LXD.StoragePool, "lxd-storage-pool", "", "LXD storage pool for VMs")
	flags.StringVar(&o.LXD.Network, "lxd-network", "", "LXD network for VMs")
	flags.StringVar(&o.StaticIP.IP, "static-ip", "", "Static IP address for VMs")
	flags.StringVar(&o.StaticIP.CIDR, "static-ip-cidr", "", "Network CIDR for static IP")
	flags.StringVar(&o.StaticIP.Gateway, "static-ip-gateway", "", "Network gateway for static IP")
	flags.StringVar(&o.NameserversRaw, "static-ip-nameservers", "", "Comma-separated list of DNS servers")

	// Note: Required flags are validated in the Validate() method
}
