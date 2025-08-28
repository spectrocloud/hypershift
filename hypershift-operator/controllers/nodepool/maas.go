package nodepool

import (
	"fmt"

	hyperv1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/openshift/hypershift/hypershift-operator/controllers/nodepool/maas"

	capimaas "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *CAPI) maasMachineTemplate(templateNameGenerator func(spec any) (string, error)) (*capimaas.MaasMachineTemplate, error) {
	nodePool := c.nodePool
	spec, err := maas.MachineTemplateSpec(nodePool)
	if err != nil {
		SetStatusCondition(&nodePool.Status.Conditions, hyperv1.NodePoolCondition{
			Type:               hyperv1.NodePoolValidMachineTemplateConditionType,
			Status:             corev1.ConditionFalse,
			Reason:             hyperv1.InvalidMAASMachineTemplate,
			Message:            err.Error(),
			ObservedGeneration: nodePool.Generation,
		})

		return nil, err
	} else {
		removeStatusCondition(&nodePool.Status.Conditions, hyperv1.NodePoolValidMachineTemplateConditionType)
	}

	templateName, err := templateNameGenerator(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to generate template name: %w", err)
	}

	template := &capimaas.MaasMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: templateName,
		},
		Spec: *spec,
	}

	return template, nil
}

