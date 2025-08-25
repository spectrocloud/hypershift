package nodepool

import (
	"fmt"

	capimaas "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *CAPI) maasMachineTemplate(templateNameGenerator func(spec any) (string, error)) (*capimaas.MaasMachineTemplate, error) {
	spec := capimaas.MaasMachineTemplateSpec{}

	// MAAS machine template spec is currently minimal since MAASNodePoolPlatform is empty
	// but we can extend this in the future if needed

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
