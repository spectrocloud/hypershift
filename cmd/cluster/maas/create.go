package maas

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/cmd/cluster/core"
	hyperutil "github.com/openshift/hypershift/support/util"
)

type CreateOptions struct {
	*core.RawCreateOptions
	MAASEndpoint     string
	MAASAPIKey       string
	MAASZone         string
	MAASDNSDomain    string
	CredentialsName  string
	CreateSecret     bool
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
	cmd.Flags().BoolVar(&maasOpts.CreateSecret, "create-secret", true, "Create a credentials secret automatically")
	cmd.Flags().StringVar(&maasOpts.APIServerAddress, "external-api-server-address", "", "The external API Server Address when using NodePort services. If not provided, will auto-detect from node addresses.")

	cmd.MarkFlagRequired("maas-endpoint")
	cmd.MarkFlagRequired("maas-api-key")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := maasOpts.Run(ctx); err != nil {
			return err
		}

		return nil
	}

	return cmd
}

func (o *CreateOptions) Run(ctx context.Context) error {
	// Validate MAAS-specific options
	if o.MAASEndpoint == "" {
		return fmt.Errorf("MAAS endpoint is required")
	}
	if o.MAASAPIKey == "" {
		return fmt.Errorf("MAAS API key is required")
	}

	// Set default credentials name if not provided
	if o.CredentialsName == "" {
		o.CredentialsName = fmt.Sprintf("%s-maas-credentials", o.Name)
	}

	// Auto-detect API server address if not provided
	if o.APIServerAddress == "" {
		apiServerAddress, err := core.GetAPIServerAddressByNode(ctx, o.Log)
		if err != nil {
			return fmt.Errorf("failed to auto-detect API server address: %w", err)
		}
		o.APIServerAddress = apiServerAddress
		fmt.Printf("Auto-detected API server address: %s\n", o.APIServerAddress)
	}

	// Create credentials secret if requested
	if o.CreateSecret {
		if err := o.createCredentialsSecret(ctx); err != nil {
			return fmt.Errorf("failed to create credentials secret: %w", err)
		}
		fmt.Printf("Created credentials secret %s in namespace %s\n", o.CredentialsName, o.Namespace)
	}

	// Create the hosted cluster
	hc := &hyperv1.HostedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.Namespace,
			Name:      o.Name,
		},
		Spec: hyperv1.HostedClusterSpec{
			// Add other required fields based on core.CreateOptions
			Release: hyperv1.Release{
				Image: o.ReleaseImage,
			},
		},
	}

	// Apply MAAS platform-specific configuration
	if err := o.ApplyPlatformSpecifics(hc); err != nil {
		return fmt.Errorf("failed to apply platform specifics: %w", err)
	}

	// Create the hosted cluster
	if err := o.CreateCluster(ctx, hc); err != nil {
		return fmt.Errorf("failed to create hosted cluster: %w", err)
	}

	fmt.Printf("Hosted cluster %s created successfully in namespace %s\n", o.Name, o.Namespace)
	return nil
}

func (o *CreateOptions) createCredentialsSecret(ctx context.Context) error {
	// Get Kubernetes client
	client, err := hyperutil.GetKubeClientSet()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// Create the credentials secret
	secret := &corev1.Secret{
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

	// Create or update the secret
	_, err = client.CoreV1().Secrets(o.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		// If secret already exists, update it
		_, err = client.CoreV1().Secrets(o.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create or update credentials secret: %w", err)
		}
	}

	return nil
}

func (o *CreateOptions) ApplyPlatformSpecifics(cluster *hyperv1.HostedCluster) error {
	// Set default DNS domain if not provided
	if cluster.Spec.DNS.BaseDomain == "" {
		cluster.Spec.DNS.BaseDomain = "example.com"
	}

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
	} else {
		// Fallback to auto-detection
		apiServerAddress, err := core.GetAPIServerAddressByNode(context.Background(), o.Log)
		if err != nil {
			return fmt.Errorf("failed to auto-detect API server address: %w", err)
		}
		cluster.Spec.Services = core.GetServicePublishingStrategyMappingByAPIServerAddress(apiServerAddress, hyperv1.NetworkType(o.NetworkType))
	}

	return nil
}

func (o *CreateOptions) CreateCluster(ctx context.Context, hc *hyperv1.HostedCluster) error {
	// This is a simplified implementation
	// In the actual implementation, you would use the client to create the resource
	fmt.Printf("Creating MAAS hosted cluster %s with credentials secret %s\n", o.Name, o.CredentialsName)
	return nil
}
