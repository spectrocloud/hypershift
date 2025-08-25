package nodepool

import (
	"fmt"

	capimaas "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func (c *CAPI) maasMachineTemplate(templateNameGenerator func(spec any) (string, error)) (*capimaas.MaasMachineTemplate, error) {
	nodePool := c.nodePool

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
	}

	// Create the template spec
	spec := capimaas.MaasMachineTemplateSpec{
		Template: capimaas.MaasMachineTemplateResource{
			Spec: maasMachineSpec,
		},
	}

	templateName, err := templateNameGenerator(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to generate template name: %w", err)
	}

	template := &capimaas.MaasMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: templateName,
		},
		Spec: spec,
	}

	return template, nil
}
