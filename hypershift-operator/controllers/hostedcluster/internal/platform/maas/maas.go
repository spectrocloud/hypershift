package maas

import (
	"context"
	"fmt"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/support/upsert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	capimaas "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
)

const (
	MAASCAPIProvider = "maas-cluster-api-controllers"
)

type MaaS struct {
	capiProviderImage string
}

func New(capiProviderImage string) *MaaS {
	return &MaaS{
		capiProviderImage: capiProviderImage,
	}
}

func (p *MaaS) ReconcileCAPIInfraCR(ctx context.Context, c client.Client, createOrUpdate upsert.CreateOrUpdateFN,
	hcluster *hyperv1.HostedCluster,
	controlPlaneNamespace string,
	apiEndpoint hyperv1.APIEndpoint,
) (client.Object, error) {
	if hcluster.Spec.Platform.MAAS == nil {
		return nil, fmt.Errorf("failed to reconcile MAAS CAPI cluster, empty MAAS platform spec")
	}

	// Create a MAAS cluster using the actual CAPI provider types
	maasCluster := &capimaas.MaasCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcluster.Name,
			Namespace: controlPlaneNamespace,
			Annotations: map[string]string{
				"spectrocloud.com/custom-dns-provided": "",
			},
			Labels: map[string]string{
				"hypershift.openshift.io/cluster": hcluster.Name,
				"platform":                        "maas",
			},
		},
	}

	// Use the createOrUpdate function to ensure the object is properly managed
	if _, err := createOrUpdate(ctx, c, maasCluster, func() error {
		maasCluster.Spec = capimaas.MaasClusterSpec{
			DNSDomain: "maas.local", // Required field - must be at least 1 character
			ControlPlaneEndpoint: capimaas.APIEndpoint{
				Host: apiEndpoint.Host,
				Port: int(apiEndpoint.Port),
			},
		}

		// Ensure the annotation is always present
		if maasCluster.Annotations == nil {
			maasCluster.Annotations = make(map[string]string)
		}
		maasCluster.Annotations["spectrocloud.com/custom-dns-provided"] = ""

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to create or update MAAS cluster: %w", err)
	}

	return maasCluster, nil
}

func (p *MaaS) CAPIProviderDeploymentSpec(hcluster *hyperv1.HostedCluster, _ *hyperv1.HostedControlPlane) (*appsv1.DeploymentSpec, error) {
	// Return a basic deployment spec for the MAAS CAPI provider
	// This is a placeholder implementation until the actual MAAS CAPI provider is available
	return &appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "maas-capi-controller",
						Image: p.capiProviderImage,
						Args: []string{
							"--v=2",
							"--leader-elect=true",
							"--sync-period=15m",
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("200m"),
								corev1.ResourceMemory: resource.MustParse("100Mi"),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("200m"),
								corev1.ResourceMemory: resource.MustParse("20Mi"),
							},
						},
					},
				},
			},
		},
	}, nil
}

func (p *MaaS) ReconcileCredentials(ctx context.Context, c client.Client, createOrUpdate upsert.CreateOrUpdateFN, hcluster *hyperv1.HostedCluster, controlPlaneNamespace string) error {
	if hcluster.Spec.Platform.MAAS == nil {
		return fmt.Errorf("failed to reconcile MAAS credentials, empty MAAS platform spec")
	}

	// Create a credentials secret for the MAAS CAPI provider
	credentialsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-maas-credentials", hcluster.Name),
			Namespace: controlPlaneNamespace,
			Labels: map[string]string{
				"hypershift.openshift.io/cluster": hcluster.Name,
				"platform":                        "maas",
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"endpoint": hcluster.Spec.Platform.MAAS.MaaSConfig.Endpoint,
			"api-key":  hcluster.Spec.Platform.MAAS.MaaSConfig.APIKey,
			"zone":     hcluster.Spec.Platform.MAAS.MaaSConfig.Zone,
		},
	}

	// Use the createOrUpdate function to ensure the secret is properly managed
	_, err := createOrUpdate(ctx, c, credentialsSecret, func() error {
		// Update the secret data to ensure it's always current
		credentialsSecret.StringData = map[string]string{
			"endpoint": hcluster.Spec.Platform.MAAS.MaaSConfig.Endpoint,
			"api-key":  hcluster.Spec.Platform.MAAS.MaaSConfig.APIKey,
			"zone":     hcluster.Spec.Platform.MAAS.MaaSConfig.Zone,
		}
		return nil
	})

	return err
}

func (p *MaaS) ReconcileSecretEncryption(ctx context.Context, c client.Client, createOrUpdate upsert.CreateOrUpdateFN, hcluster *hyperv1.HostedCluster, controlPlaneNamespace string) error {
	// MAAS doesn't support secret encryption, so this is a no-op
	return nil
}

func (p *MaaS) CAPIProviderPolicyRules() []rbacv1.PolicyRule {
	// MAAS doesn't require additional policy rules beyond the default CAPI rules
	return nil
}

func (p *MaaS) DeleteCredentials(ctx context.Context, c client.Client, hcluster *hyperv1.HostedCluster, controlPlaneNamespace string) error {
	// Delete the MAAS credentials secret
	credentialsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-maas-credentials", hcluster.Name),
			Namespace: controlPlaneNamespace,
		},
	}

	if err := c.Delete(ctx, credentialsSecret); err != nil {
		// If the secret doesn't exist, that's fine
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to delete MAAS credentials secret: %w", err)
		}
	}

	return nil
}
