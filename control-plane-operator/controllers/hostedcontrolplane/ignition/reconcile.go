package ignition

import (
	"bytes"
	"fmt"

	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/manifests"
	"github.com/openshift/hypershift/support/api"
	"github.com/openshift/hypershift/support/config"

	configv1 "github.com/openshift/api/config/v1"
	mcfgv1 "github.com/openshift/api/machineconfiguration/v1"
	operatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	jsonserializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/clarketm/json"
	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
)

const (
	ignitionConfigKey = "config"
	ignitionVersion   = "3.2.0"
)

var (
	defaultMachineConfigLabels = map[string]string{
		"machineconfiguration.openshift.io/role": "worker",
	}

	defaultIgnitionConfigMapLabels = map[string]string{
		"hypershift.openshift.io/core-ignition-config": "true",
	}
)

func ReconcileFIPSIgnitionConfig(cm *corev1.ConfigMap, ownerRef config.OwnerRef, fipsEnabled bool) error {
	machineConfig := manifests.MachineConfigFIPS()
	SetMachineConfigLabels(machineConfig)
	machineConfig.Spec.FIPS = fipsEnabled
	return reconcileMachineConfigIgnitionConfigMap(cm, machineConfig, ownerRef)
}

func ReconcileWorkerSSHIgnitionConfig(cm *corev1.ConfigMap, ownerRef config.OwnerRef, sshKey string) error {
	machineConfig := manifests.MachineConfigWorkerSSH()
	SetMachineConfigLabels(machineConfig)
	serializedConfig, err := workerSSHConfig(sshKey)
	if err != nil {
		return fmt.Errorf("failed to serialize ignition config: %w", err)
	}
	machineConfig.Spec.Config.Raw = serializedConfig
	return reconcileMachineConfigIgnitionConfigMap(cm, machineConfig, ownerRef)
}

func ReconcileImageSourceMirrorsIgnitionConfigFromIDMS(cm *corev1.ConfigMap, ownerRef config.OwnerRef, imageDigestMirrorSet *configv1.ImageDigestMirrorSet) error {
	return reconcileImageContentTypeIgnitionConfigMap(cm, imageDigestMirrorSet, ownerRef)
}

func ReconcileMAASIgnitionConfig(cm *corev1.ConfigMap, ownerRef config.OwnerRef) error {
	machineConfig := manifests.MachineConfigMAAS()
	SetMachineConfigLabels(machineConfig)

	// Generate MAAS-specific ignition content
	serializedConfig, err := maasIgnitionConfig()
	if err != nil {
		return fmt.Errorf("failed to serialize MAAS ignition config: %w", err)
	}
	machineConfig.Spec.Config.Raw = serializedConfig

	return reconcileMachineConfigIgnitionConfigMap(cm, machineConfig, ownerRef)
}

func workerSSHConfig(sshKey string) ([]byte, error) {
	config := &igntypes.Config{}
	config.Ignition.Version = ignitionVersion
	config.Passwd = igntypes.Passwd{
		Users: []igntypes.PasswdUser{
			{
				Name: "core",
			},
		},
	}
	if len(sshKey) > 0 {
		config.Passwd.Users[0].SSHAuthorizedKeys = []igntypes.SSHAuthorizedKey{
			igntypes.SSHAuthorizedKey(sshKey),
		}
	}
	return serializeIgnitionConfig(config)
}

func maasIgnitionConfig() ([]byte, error) {
	config := &igntypes.Config{}
	config.Ignition.Version = ignitionVersion

	// Add custom user configuration for MAAS
	config.Passwd = igntypes.Passwd{
		Users: []igntypes.PasswdUser{
			{
				Name:         "spectro",
				Groups:       []igntypes.Group{"sudo"},
				Shell:        ptr.To("/bin/bash"),
				PasswordHash: ptr.To("$6$salt$JmHhpqORjPckABM.DZyXAntcxWnkBL/hC5B8xiwweGUYepl2N0AqVnkfJWMv9F0xFAIIz2siruaP7J2qnSyWH/"),
				SSHAuthorizedKeys: []igntypes.SSHAuthorizedKey{
					"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDCzobpcF70X7oMUzK6xT2JgO6O57vgiT/9FepL7003hDw1QIxycwU9gmKhxBhJ69140RVSZi6IMmYN26xCJNcB/1cGejBRog59u4YNCcCUTzPz+jU2ZtjPJtYSFRMlR7qMI7aGFdx1rOxo8HmL9bV4ks+zWhtPTSvsk9zzte+XD0xIecoK/aewXQ7pEyfxXu/YQzAg+uUqrlOq+X4SRXGaWA02dfwQjr0kDQHFMtDZjyMmyXuNvNgPKBar5RkXeomdSw4IPuXzaU84RJfxGzF3SOYIqPWNfZCIPWPWOl7zBHnXg1+JI1LQFTs4sqpamML6mv+lMkvhJfX8CXFkzOmAteTjRYIi6f59QcSgfUeahb8R64CsAdqYKirCbIz9pz+UGM4Hc1mndUM/vf+UejidSQ+npxVP1nolpR2jLmLzad/9yracHikHTTf3WdjHM1aW/RtbY2y/Km/9ObVRw8agKVsu45sdN0KFI981E4Bb/1lDvxzSI32FOhLUgOW//SMFa/JQj78JgXkCZXCuA9f5U+DLFo6s7FAjsiFXyX6LMs/xO1jw3CkmgxMNfU7rc4Vj63xBYYWJsTGQsCDintodsHFZn/IOefJCCOQ0OMJxRSZqLu/Fp5Sd6iO6YsK3VqCh1RRYza1I81G9yBfUhIru87UPHpub/XqFh3/hWf19Jw== spectro2024",
				},
			},
		},
	}

	// Add MAAS-specific storage configuration
	config.Storage = igntypes.Storage{
		Disks: []igntypes.Disk{
			{
				Device: "/dev/sda",
				Partitions: []igntypes.Partition{
					{
						Number:             5,
						ShouldExist:        ptr.To(false),
						WipePartitionEntry: ptr.To(true),
					},
				},
			},
		},
	}

	return serializeIgnitionConfig(config)
}

func serializeIgnitionConfig(cfg *igntypes.Config) ([]byte, error) {
	jsonBytes, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("error marshaling ignition config: %w", err)
	}
	return jsonBytes, nil
}

func SetMachineConfigLabels(mc *mcfgv1.MachineConfig) {
	if mc.Labels == nil {
		mc.Labels = map[string]string{}
	}
	for k, v := range defaultMachineConfigLabels {
		mc.Labels[k] = v
	}
}

func reconcileImageContentTypeIgnitionConfigMap(cm *corev1.ConfigMap, imageContentType client.Object, ownerRef config.OwnerRef) error {
	scheme := runtime.NewScheme()
	err := operatorv1alpha1.Install(scheme)
	if err != nil {
		return err
	}
	err = configv1.Install(scheme)
	if err != nil {
		return err
	}
	yamlSerializer := jsonserializer.NewSerializerWithOptions(
		jsonserializer.DefaultMetaFactory, scheme, scheme,
		jsonserializer.SerializerOptions{Yaml: true, Pretty: true, Strict: true})
	imageContentTypeBytesBuffer := bytes.NewBuffer([]byte{})
	if err := yamlSerializer.Encode(imageContentType, imageContentTypeBytesBuffer); err != nil {
		return fmt.Errorf("failed to serialize image content type policy: %w", err)
	}
	return ReconcileIgnitionConfigMap(cm, imageContentTypeBytesBuffer.String(), ownerRef)
}

func reconcileMachineConfigIgnitionConfigMap(cm *corev1.ConfigMap, mc *mcfgv1.MachineConfig, ownerRef config.OwnerRef) error {
	buf := &bytes.Buffer{}
	mc.APIVersion = mcfgv1.SchemeGroupVersion.String()
	mc.Kind = "MachineConfig"
	if err := api.YamlSerializer.Encode(mc, buf); err != nil {
		return fmt.Errorf("failed to serialize machine config %s: %w", cm.Name, err)
	}
	return ReconcileIgnitionConfigMap(cm, buf.String(), ownerRef)
}

func ReconcileIgnitionConfigMap(cm *corev1.ConfigMap, content string, ownerRef config.OwnerRef) error {
	ownerRef.ApplyTo(cm)
	if cm.Labels == nil {
		cm.Labels = map[string]string{}
	}
	for k, v := range defaultIgnitionConfigMapLabels {
		cm.Labels[k] = v
	}
	cm.Data = map[string]string{
		ignitionConfigKey: content,
	}
	return nil
}
