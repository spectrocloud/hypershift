package maas

import (
	"context"
	"fmt"
	"os"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/support/images"
	"github.com/openshift/hypershift/support/upsert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		// Get DNS domain from MAAS platform spec or use a default
		dnsDomain := "maas.local"
		if hcluster.Spec.Platform.MAAS.DNSDomain != "" {
			dnsDomain = hcluster.Spec.Platform.MAAS.DNSDomain
		}

		maasCluster.Spec = capimaas.MaasClusterSpec{
			DNSDomain: dnsDomain,
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
	// Check for environment variable override
	image := p.capiProviderImage
	if envImage := os.Getenv(images.MAASCAPIProviderEnvVar); len(envImage) > 0 {
		image = envImage
	}

	// Return a deployment spec for the MAAS CAPI provider with proper credential mounting
	return &appsv1.DeploymentSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "maas-capi-controller",
						Image: image,
						Args: []string{
							"--v=2",
							"--leader-elect=true",
							"--sync-period=15m",
							"--namespace=$(NAMESPACE)",
						},
						Env: []corev1.EnvVar{
							{
								Name: "NAMESPACE",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							},
							{
								Name: "MAAS_ENDPOINT",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: hcluster.Spec.Platform.MAAS.IdentityRef.Name,
										},
										Key: "MAAS_ENDPOINT",
									},
								},
							},
							{
								Name: "MAAS_API_KEY",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: hcluster.Spec.Platform.MAAS.IdentityRef.Name,
										},
										Key: "MAAS_API_KEY",
									},
								},
							},
							{
								Name:  "MAAS_ZONE",
								Value: hcluster.Spec.Platform.MAAS.Zone,
							},
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

	// Get the referenced credentials secret
	credentialsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcluster.Spec.Platform.MAAS.IdentityRef.Name,
			Namespace: hcluster.Namespace,
		},
	}

	if err := c.Get(ctx, client.ObjectKeyFromObject(credentialsSecret), credentialsSecret); err != nil {
		return fmt.Errorf("failed to get MAAS credentials secret: %w", err)
	}

	// Validate that the secret contains the required keys
	requiredKeys := []string{"MAAS_ENDPOINT", "MAAS_API_KEY"}
	for _, key := range requiredKeys {
		if _, exists := credentialsSecret.Data[key]; !exists {
			return fmt.Errorf("MAAS credentials secret is missing required key: %s", key)
		}
	}

	// Create a copy of the secret in the control plane namespace
	// Use the same name as referenced in IdentityRef so capi-provider can find it
	controlPlaneSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcluster.Spec.Platform.MAAS.IdentityRef.Name,
			Namespace: controlPlaneNamespace,
			Labels: map[string]string{
				"hypershift.openshift.io/cluster": hcluster.Name,
				"platform":                        "maas",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: credentialsSecret.Data,
	}

	// Check if the secret already exists and has the same data
	existingSecret := &corev1.Secret{}
	err := c.Get(ctx, client.ObjectKeyFromObject(controlPlaneSecret), existingSecret)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get existing MAAS credentials secret: %w", err)
		}
		// Secret doesn't exist, create it
		controlPlaneSecret.Data = credentialsSecret.Data
		_, err = createOrUpdate(ctx, c, controlPlaneSecret, func() error {
			return nil
		})
	} else {
		// Secret exists, check if data has changed
		dataChanged := existingSecret.Data == nil ||
			string(existingSecret.Data["MAAS_ENDPOINT"]) != string(credentialsSecret.Data["MAAS_ENDPOINT"]) ||
			string(existingSecret.Data["MAAS_API_KEY"]) != string(credentialsSecret.Data["MAAS_API_KEY"])

		if dataChanged {
			// Data has changed, update the secret
			controlPlaneSecret.Data = credentialsSecret.Data
			_, err = createOrUpdate(ctx, c, controlPlaneSecret, func() error {
				return nil
			})
		}
		// If data hasn't changed, do nothing
	}

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
