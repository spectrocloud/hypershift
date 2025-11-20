package maas

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/cmd/cluster/core"
)

type CreateOptions struct {
	*core.RawCreateOptions
	MAASEndpoint     string
	MAASAPIKey       string
	MAASZone         string
	MAASDNSDomain    string
	CredentialsName  string
	APIServerAddress string // Add support for external API server address
}

func NewCreateCommand(opts *core.RawCreateOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "maas",
		Short:        "Creates a MAAS HostedCluster",
		SilenceUsage: true,
	}

	maasOpts := &CreateOptions{
		RawCreateOptions: opts,
	}

	cmd.Flags().StringVar(&maasOpts.MAASEndpoint, "maas-endpoint", "", "MAAS API endpoint URL")
	cmd.Flags().StringVar(&maasOpts.MAASAPIKey, "maas-api-key", "", "MAAS API key for authentication")
	cmd.Flags().StringVar(&maasOpts.MAASZone, "maas-zone", "", "MAAS zone where the cluster will be deployed")
	cmd.Flags().StringVar(&maasOpts.MAASDNSDomain, "maas-dns-domain", "", "DNS domain for the MAAS cluster")
	cmd.Flags().StringVar(&maasOpts.CredentialsName, "credentials-name", "", "Name of the credentials secret (defaults to <cluster-name>-maas-credentials)")
	cmd.Flags().StringVar(&maasOpts.APIServerAddress, "external-api-server-address", "", "The external API Server Address when using NodePort services. If not provided, will auto-detect from node addresses.")

	cmd.MarkFlagRequired("maas-endpoint")
	cmd.MarkFlagRequired("maas-api-key")

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

// Validate implements core.PlatformValidator
func (o *CreateOptions) Validate(ctx context.Context, opts *core.CreateOptions) (core.PlatformCompleter, error) {
	// Validate MAAS-specific options
	if o.MAASEndpoint == "" {
		return nil, fmt.Errorf("MAAS endpoint is required")
	}
	if o.MAASAPIKey == "" {
		return nil, fmt.Errorf("MAAS API key is required")
	}

	// Set default credentials name if not provided
	if o.CredentialsName == "" {
		o.CredentialsName = fmt.Sprintf("%s-maas-credentials", opts.Name)
	}

	return o, nil
}

// Complete implements core.PlatformCompleter
func (o *CreateOptions) Complete(ctx context.Context, opts *core.CreateOptions) (core.Platform, error) {
	// Auto-detect API server address if not provided
	if o.APIServerAddress == "" {
		apiServerAddress, err := core.GetAPIServerAddressByNode(ctx, opts.Log)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-detect API server address: %w", err)
		}
		o.APIServerAddress = apiServerAddress
		opts.Log.Info("Auto-detected API server address", "address", o.APIServerAddress)
	}

	return o, nil
}

// ApplyPlatformSpecifics implements core.Platform
func (o *CreateOptions) ApplyPlatformSpecifics(cluster *hyperv1.HostedCluster) error {
	// Configure platform spec
	cluster.Spec.Platform = hyperv1.PlatformSpec{
		Type: hyperv1.MAASPlatform,
		MAAS: &hyperv1.MAASPlatformSpec{
			IdentityRef: hyperv1.MAASIdentityReference{
				Name: o.CredentialsName,
			},
			DNSDomain: o.MAASDNSDomain,
			Zone:      o.MAASZone,
		},
	}

	// Configure services with NodePort and detected API server address
	if o.APIServerAddress != "" {
		cluster.Spec.Services = core.GetServicePublishingStrategyMappingByAPIServerAddress(o.APIServerAddress, hyperv1.NetworkType(o.NetworkType))
	}

	return nil
}

// GenerateNodePools implements core.Platform
func (o *CreateOptions) GenerateNodePools(constructor core.DefaultNodePoolConstructor) []*hyperv1.NodePool {
	nodePool := constructor(hyperv1.MAASPlatform, "")
	nodePool.Spec.Platform.MAAS = &hyperv1.MAASNodePoolPlatform{
		IdentityRef: hyperv1.MAASIdentityReference{
			Name: o.CredentialsName,
		},
		Zone: o.MAASZone,
	}
	return []*hyperv1.NodePool{nodePool}
}

// GenerateResources implements core.Platform - generates secrets for render mode
func (o *CreateOptions) GenerateResources() ([]crclient.Object, error) {
	var objects []crclient.Object

	// Generate MAAS credentials secret if credentials are provided
	// This will be included in render output when --render-sensitive is used
	if o.MAASEndpoint != "" && o.MAASAPIKey != "" {
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      o.CredentialsName,
				Namespace: o.Namespace,
				Labels: map[string]string{
					"hypershift.openshift.io/cluster": o.Name,
					"platform":                        "maas",
				},
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				"MAAS_ENDPOINT": o.MAASEndpoint,
				"MAAS_API_KEY":  o.MAASAPIKey,
			},
		}
		objects = append(objects, secret)
	}

	return objects, nil
}

// Ensure CreateOptions implements the required interfaces
var _ core.PlatformValidator = (*CreateOptions)(nil)
var _ core.PlatformCompleter = (*CreateOptions)(nil)
var _ core.Platform = (*CreateOptions)(nil)
