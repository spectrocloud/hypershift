package maas

import (
	"context"
	"errors"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/cmd/cluster/core"

	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"
)

type RawCreateOptions struct {
	Endpoint         string
	APIKey           string
	Zone             string
	APIServerAddress string
}

type validatedCreateOptions struct {
	*RawCreateOptions
}

type ValidatedCreateOptions struct {
	*validatedCreateOptions
}

type completedCreateOptions struct {
	*ValidatedCreateOptions
}

type CreateOptions struct {
	*completedCreateOptions
}

func (o *RawCreateOptions) Validate(ctx context.Context, opts *core.CreateOptions) (core.PlatformCompleter, error) {
	if err := validateMAASOptions(o); err != nil {
		return nil, err
	}

	return &ValidatedCreateOptions{
		validatedCreateOptions: &validatedCreateOptions{
			RawCreateOptions: o,
		},
	}, nil
}

func (o *ValidatedCreateOptions) Complete(ctx context.Context, opts *core.CreateOptions) (core.Platform, error) {
	return &CreateOptions{
		completedCreateOptions: &completedCreateOptions{
			ValidatedCreateOptions: o,
		},
	}, nil
}

func (o *CreateOptions) ApplyPlatformSpecifics(cluster *hyperv1.HostedCluster) error {
	cluster.Spec.Platform.Type = hyperv1.MAASPlatform
	cluster.Spec.Platform.MAAS = &hyperv1.MAASPlatformSpec{
		MaaSConfig: hyperv1.MaaSConfig{
			Endpoint: o.RawCreateOptions.Endpoint,
			APIKey:   o.RawCreateOptions.APIKey,
			Zone:     o.RawCreateOptions.Zone,
		},
	}

	// Set services for MAAS platform (similar to none platform)
	if cluster.Spec.Services == nil {
		if o.RawCreateOptions.APIServerAddress != "" {
			// Use NodePort with specific API server address (like none platform)
			cluster.Spec.Services = core.GetServicePublishingStrategyMappingByAPIServerAddress(o.RawCreateOptions.APIServerAddress, cluster.Spec.Networking.NetworkType)
		} else {
			// Fallback to LoadBalancer (default behavior)
			cluster.Spec.Services = core.GetIngressServicePublishingStrategyMapping(cluster.Spec.Networking.NetworkType, false)
		}
	}

	return nil
}

func (o *CreateOptions) GenerateNodePools(defaultNodePool core.DefaultNodePoolConstructor) []*hyperv1.NodePool {
	nodePool := defaultNodePool(hyperv1.MAASPlatform, "")

	// Set upgrade type if not set (similar to none platform)
	if nodePool.Spec.Management.UpgradeType == "" {
		nodePool.Spec.Management.UpgradeType = hyperv1.UpgradeTypeInPlace
	}

	return []*hyperv1.NodePool{nodePool}
}

func (o *CreateOptions) GenerateResources() ([]crclient.Object, error) {
	// MAAS doesn't require additional infrastructure resources
	return []crclient.Object{}, nil
}

func validateMAASOptions(opts *RawCreateOptions) error {
	if opts.Endpoint == "" {
		return errors.New("--maas-endpoint is required")
	}
	if opts.APIKey == "" {
		return errors.New("--maas-api-key is required")
	}
	return nil
}

func NewCreateCommand(opts *core.RawCreateOptions) *cobra.Command {
	maasOpts := &RawCreateOptions{}

	cmd := &cobra.Command{
		Use:          "maas",
		Short:        "Creates basic functional HostedCluster resources on MAAS",
		SilenceUsage: true,
	}

	// Add MaaS-specific flags
	cmd.Flags().StringVar(&maasOpts.Endpoint, "maas-endpoint", "", "MaaS API endpoint")
	cmd.Flags().StringVar(&maasOpts.APIKey, "maas-api-key", "", "MaaS API key")
	cmd.Flags().StringVar(&maasOpts.Zone, "maas-zone", "", "MaaS zone")
	cmd.Flags().StringVar(&maasOpts.APIServerAddress, "external-api-server-address", "", "The external API Server Address for MaaS platform (uses NodePort instead of LoadBalancer)")

	_ = cmd.MarkFlagRequired("maas-endpoint")
	_ = cmd.MarkFlagRequired("maas-api-key")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if opts.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
			defer cancel()
		}

		if err := core.CreateCluster(ctx, opts, maasOpts); err != nil {
			opts.Log.Error(err, "Failed to create cluster")
			return err
		}
		return nil
	}

	return cmd
}
