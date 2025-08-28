package maas

import (
	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	capimaas "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	"k8s.io/utils/ptr"
)

// MachineTemplateSpec creates a MAAS machine template specification for the given NodePool
func MachineTemplateSpec(nodePool *hyperv1.NodePool) (*capimaas.MaasMachineTemplateSpec, error) {
	// Create MAAS machine spec with required fields
	maasMachineSpec := capimaas.MaasMachineSpec{
		// Required fields
		Image:         "ubuntu/focal", // Default image - should be configurable
		MinCPU:        ptr.To(1),      // Default minimum CPU
		MinMemoryInMB: ptr.To(1024),   // Default minimum memory in MB
	}

	// Override with NodePool platform configuration if specified
	if nodePool.Spec.Platform.MAAS != nil {
		if nodePool.Spec.Platform.MAAS.Image != "" {
			maasMachineSpec.Image = nodePool.Spec.Platform.MAAS.Image
		}
		if nodePool.Spec.Platform.MAAS.MinCPU != nil {
			maasMachineSpec.MinCPU = ptr.To(int(*nodePool.Spec.Platform.MAAS.MinCPU))
		}
		if nodePool.Spec.Platform.MAAS.MinMemory != nil {
			maasMachineSpec.MinMemoryInMB = ptr.To(int(*nodePool.Spec.Platform.MAAS.MinMemory))
		}
		if nodePool.Spec.Platform.MAAS.Tags != nil {
			maasMachineSpec.Tags = nodePool.Spec.Platform.MAAS.Tags
		}
		if nodePool.Spec.Platform.MAAS.ResourcePool != "" {
			maasMachineSpec.ResourcePool = ptr.To(nodePool.Spec.Platform.MAAS.ResourcePool)
		}
		if nodePool.Spec.Platform.MAAS.FailureDomain != "" {
			maasMachineSpec.FailureDomain = ptr.To(nodePool.Spec.Platform.MAAS.FailureDomain)
		}
	}

	// Create the template spec
	spec := capimaas.MaasMachineTemplateSpec{
		Template: capimaas.MaasMachineTemplateResource{
			Spec: maasMachineSpec,
		},
	}

	return &spec, nil
}
