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
		// Map image
		if nodePool.Spec.Platform.MAAS.Image != "" {
			maasMachineSpec.Image = nodePool.Spec.Platform.MAAS.Image
		}
		
		// Map CPU/Memory requirements
		if nodePool.Spec.Platform.MAAS.MinCPU != nil {
			maasMachineSpec.MinCPU = ptr.To(int(*nodePool.Spec.Platform.MAAS.MinCPU))
		}
		if nodePool.Spec.Platform.MAAS.MinMemory != nil {
			maasMachineSpec.MinMemoryInMB = ptr.To(int(*nodePool.Spec.Platform.MAAS.MinMemory))
		}
		
		// Map failure domain (prefer FailureDomain over Zone)
		if nodePool.Spec.Platform.MAAS.FailureDomain != "" {
			maasMachineSpec.FailureDomain = ptr.To(nodePool.Spec.Platform.MAAS.FailureDomain)
		} else if nodePool.Spec.Platform.MAAS.Zone != "" {
			// Map Zone to FailureDomain if FailureDomain not specified
			maasMachineSpec.FailureDomain = ptr.To(nodePool.Spec.Platform.MAAS.Zone)
		}
		
		// Map resource pool
		if nodePool.Spec.Platform.MAAS.ResourcePool != "" {
			maasMachineSpec.ResourcePool = ptr.To(nodePool.Spec.Platform.MAAS.ResourcePool)
		}
		
		// Map tags
		if len(nodePool.Spec.Platform.MAAS.Tags) > 0 {
			maasMachineSpec.Tags = nodePool.Spec.Platform.MAAS.Tags
		}
		
		// TODO: Add support for MinDiskSize, LXD, and StaticIP when available in HyperShift API
		// These fields are defined in the API but not yet available in the controller
	}

	// Create the template spec
	spec := capimaas.MaasMachineTemplateSpec{
		Template: capimaas.MaasMachineTemplateResource{
			Spec: maasMachineSpec,
		},
	}

	return &spec, nil
}
