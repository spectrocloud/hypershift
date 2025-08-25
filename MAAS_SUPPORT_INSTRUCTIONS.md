# Adding MaaS Support to HyperShift

This document outlines the steps to add MaaS (Metal as a Service) support to HyperShift, similar to existing cloud providers like AWS, Azure, and OpenStack.

## Overview

MaaS support has been successfully integrated into HyperShift using the [spectrocloud/cluster-api-provider-maas](https://github.com/spectrocloud/cluster-api-provider-maas) CAPI provider. The integration follows the same pattern as other platforms:

1. **API Types**: Added MaaS-specific configuration to the HostedCluster API
2. **Platform Implementation**: Created a MaaS platform implementation that implements the `Platform` interface
3. **CLI Commands**: Added MaaS-specific CLI commands for cluster creation and destruction
4. **CAPI Integration**: Integrated with the MaaS CAPI provider for infrastructure management
5. **Automatic Annotations**: Automatically adds required annotations for proper CAPI provider operation
6. **Resource Management**: Proper resource limits and requests for the CAPI provider deployment
7. **Sync Period Configuration**: Configurable sync period for controller reconciliation

## What Was Added

### 1. API Types (`api/hypershift/v1beta1/maas.go`)

Added MaaS platform configuration to the HostedCluster API:

```go
// MAASPlatformSpec specifies configuration for clusters running on MaaS (Metal as a Service).
type MAASPlatformSpec struct {
    // maasConfig specifies the MaaS configuration for the cluster.
    // +required
    MaaSConfig MaaSConfig `json:"maasConfig"`
}

// MaaSConfig specifies the MaaS API configuration.
type MaaSConfig struct {
    // endpoint is the MaaS API endpoint URL.
    // +required
    Endpoint string `json:"endpoint"`
    
    // apiKey is the MaaS API key for authentication.
    // +required
    APIKey string `json:"apiKey"`
    
    // zone is the MaaS zone where the cluster will be deployed.
    // +optional
    Zone string `json:"zone,omitempty"`
    
    // dnsDomain is the DNS domain for the MAAS cluster.
    // +optional
    DNSDomain string `json:"dnsDomain,omitempty"`
}
```

**Note**: MaaS does not use regions like AWS or Azure. It only uses zones for machine placement.

### **NEW: Configurable DNS Domain Support**

The MAAS platform now supports configurable DNS domains per cluster. Users can specify a custom DNS domain in their HostedCluster configuration:

```yaml
apiVersion: hypershift.openshift.io/v1beta1
kind: HostedCluster
metadata:
  name: my-maas-cluster
spec:
  platform:
    type: MAAS
    maas:
      maasConfig:
        endpoint: "http://maas.example.com/MAAS"
        apiKey: "your-api-key"
        zone: "zone1"
        dnsDomain: "custom.maas.local"  # Optional: custom DNS domain
```

**Benefits of Configurable DNS Domain:**
- ✅ **Per-cluster customization** - Different DNS domains for different environments
- ✅ **Multi-tenant support** - Each tenant can have their own DNS domain
- ✅ **Environment isolation** - Separate domains for dev, staging, production
- ✅ **Backward compatibility** - Falls back to "maas.local" if not specified

**Default Behavior:**
- If `dnsDomain` is not specified, the platform uses `"maas.local"` as the default
- The DNS domain is used by the MAAS CAPI provider for cluster networking configuration
- This field is required by the CRD validation but automatically handled by the controller

### 2. Admission Webhook Validation Updates

Updated the admission webhook validation to recognize `MAAS` as a valid platform type:

#### HostedCluster Validation (`api/hypershift/v1beta1/hostedcluster_types.go`)
```go
// +openshift:validation:FeatureGateAwareEnum:featureGate="",enum=AWS;Azure;IBMCloud;KubeVirt;Agent;PowerVS;None;MAAS
// +openshift:validation:FeatureGateAwareEnum:featureGate=OpenStack,enum=AWS;Azure;IBMCloud;KubeVirt;Agent;PowerVS;None;OpenStack
// +openshift:validation:FeatureGateAwareEnum:featureGate=MAAS,enum=AWS;Azure;IBMCloud;KubeVirt;Agent;PowerVS;None;OpenStack;MAAS
```

#### NodePool Validation (`api/hypershift/v1beta1/nodepool_types.go`)
```go
// +openshift:validation:FeatureGateAwareEnum:featureGate="",enum=AWS;Azure;IBMCloud;KubeVirt;Agent;PowerVS;None;MAAS
// +openshift:validation:FeatureGateAwareEnum:featureGate=OpenStack,enum=AWS;Azure;IBMCloud;KubeVirt;Agent;PowerVS;None;OpenStack
// +openshift:validation:FeatureGateAwareEnum:featureGate=MAAS,enum=AWS;Azure;IBMCloud;KubeVirt;Agent;PowerVS;None;OpenStack;MAAS
```

#### MAAS NodePool Platform Support
```go
// maas specifies the configuration used when using MaaS platform.
// +optional
// +openshift:enable:FeatureGate=MAAS
MAAS *MAASNodePoolPlatform `json:"maas,omitempty"`

// MAASNodePoolPlatform specifies the configuration for MaaS platform.

type MAASNodePoolPlatform struct {
    // machineType specifies the type of MAAS machine to use for the nodes.
    // This corresponds to the MAAS machine type/tag that will be used for node selection.
    // +optional
    // +kubebuilder:validation:MaxLength=255
    MachineType string `json:"machineType,omitempty"`

    // zone specifies the MAAS zone where the nodes will be deployed.
    // If not specified, nodes will be deployed in any available zone.
    // +optional
    // +kubebuilder:validation:MaxLength=255
    Zone string `json:"zone,omitempty"`

    // tags specifies additional MAAS tags to apply to the nodes for filtering and organization.
    // +optional
    // +kubebuilder:validation:MaxItems=10
    Tags []string `json:"tags,omitempty"`

    // resourcePool specifies the MAAS resource pool to use for node allocation.
    // +optional
    // +kubebuilder:validation:MaxLength=255
    ResourcePool string `json:"resourcePool,omitempty"`

    // minCpu specifies the minimum CPU count required for the nodes.
    // +optional
    // +kubebuilder:validation:Minimum=1
    MinCPU *int32 `json:"minCpu,omitempty"`

    // minMemory specifies the minimum memory in MB required for the nodes.
    // +optional
    // +kubebuilder:validation:Minimum=1024
    MinMemory *int32 `json:"minMemory,omitempty"`
}

**Important Notes**:

1. **Struct Location**: The `MAASNodePoolPlatform` struct must be defined in `api/hypershift/v1beta1/nodepool_types.go` where it is referenced, not in a separate `maas.go` file. This is because the `controller-gen` tool that generates CRDs has limitations in resolving cross-file type references within the same package.

2. **Properties**: The struct now includes meaningful configuration options:
   - `MachineType`: For specifying MAAS machine type/tag selection
   - `Zone`: For zone placement configuration
   - `Tags`: For custom filtering and organization
   - `ResourcePool`: For resource pool allocation
   - `MinCPU`/`MinMemory`: For resource requirements

3. **CRD Generation**: With these properties, the generated NodePool CRD will show detailed configuration options instead of just `type: object`, making it much more useful for users.
const (
    // ... existing constants ...
    
    // MAASPlatformType represents MaaS (Metal as a Service) infrastructure.
    MAASPlatformType PlatformType = "MAAS"
)
```

**Why This Was Necessary**: The `+openshift:validation:FeatureGateAwareEnum` annotations in HyperShift API types are OpenShift-specific and are NOT processed by the standard `controller-gen` tool. This means the generated CRDs were missing the `MAAS` enum value, causing admission webhook validation failures.

**The Solution**: By adding MAAS to the OpenShift API vendor code, it now gets processed by `controller-gen` automatically, ensuring that:

1. **CRDs are generated correctly** with MAAS in the platform type enum
2. **Admission webhooks work properly** for MAAS platform validation
3. **The operator can create HostedControlPlane resources** with MAAS platform type

## **NEW: Full CAPI Integration (Option 1)**

We have now implemented **full CAPI integration** for MaaS, following the same pattern as AWS, Azure, and OpenStack platforms. This provides:

### **Benefits of Full CAPI Integration**
- **Type safety** when working with MaaSCluster objects
- **Proper CRD installation** during HyperShift setup
- **Consistency** with how other platforms are handled
- **Better error handling** and validation

### **What Was Added for Full CAPI Integration**

#### **1. Scheme Registration (`support/api/scheme.go`)**
```go
// Added import
capimaas "github.com/spectrocloud/cluster-api-provider-maas/api/v1beta1"

// Added to init() function
_ = capimaas.AddToScheme(scheme)
```

#### **2. Managed Resources (`support/util/resources.go`)**
```go
// Added MaaS resources
MaaSResources = []client.Object{
    &capimaas.MaasCluster{},
}

// Added to platform switch statement
case strings.EqualFold(platform, string(hyperv1.MAASPlatform)):
    managedResources = append(managedResources, MaaSResources...)
```

#### **3. CAPI CRD Assets (`cmd/install/assets/cluster-api-provider-maas/`)**
Created `infrastructure.cluster.x-k8s.io_maasclusters.yaml` with the MaaSCluster CRD definition.

#### **4. Assets Registration (`cmd/install/assets/assets.go`)**
```go
"cluster-api-provider-maas/infrastructure.cluster.x-k8s.io_maasclusters.yaml": "v1beta1"
```

#### **5. Updated Platform Implementation**
The MaaS platform implementation now uses **typed objects** instead of unstructured ones:

```go
// Before (unstructured approach)
maasCluster := &unstructured.Unstructured{}
maasCluster.SetGroupVersionKind(schema.GroupVersionKind{...})

// After (typed approach)
maasCluster := &capimaas.MaasCluster{
    ObjectMeta: metav1.ObjectMeta{
        Name:      hcluster.Name,
        Namespace: controlPlaneNamespace,
    },
}
maasCluster.Spec = capimaas.MaasClusterSpec{
    DNSDomain: "maas.local",
    ControlPlaneEndpoint: capimaas.APIEndpoint{
        Host: apiEndpoint.Host,
        Port: int(apiEndpoint.Port),
    },
}
```

### **How CAPI CRDs Are Installed**

1. **CRD Assets are Embedded**: MaaS CAPI CRDs are stored in `cmd/install/assets/cluster-api-provider-maas/` and embedded into the HyperShift binary
2. **Installation**: When you run `make install`, these embedded CRDs are extracted and applied to the cluster
3. **Scheme Registration**: MaaS CAPI provider types are registered in the operator's scheme via `capimaas.AddToScheme(scheme)`
4. **Resource Management**: The operator uses these registered types to track and manage MaaSCluster resources during reconciliation

### **Alternative Approach (Option 2)**

If you prefer to avoid full CAPI integration, you can use the **unstructured approach**:
- Use `unstructured.Unstructured` objects instead of typed `capimaas.MaasCluster`
- Avoid the need to register MaaS CAPI types in the scheme
- But lose type safety and consistency with other platforms

## **NEW: Automatic Annotation Support**

The MaaS platform implementation now automatically adds the required `spectrocloud.com/custom-dns-provided=""` annotation to all MaaSCluster resources. This is essential for proper operation of the `capi-provider-maas`.

### **Implementation Details**

In the `ReconcileCAPIInfraCR` function, the annotation is automatically added:

```go
func (p *MaaS) ReconcileCAPIInfraCR(ctx context.Context, c client.Client, createOrUpdate upsert.CreateOrUpdateFN,
    hcluster *hyperv1.HostedCluster,
    controlPlaneNamespace string,
    apiEndpoint hyperv1.APIEndpoint,
) (client.Object, error) {
    // ... validation code ...
    
    // Create a MAAS cluster using the actual CAPI provider types
    maasCluster := &capimaas.MaasCluster{
        ObjectMeta: metav1.ObjectMeta{
            Name:      hcluster.Name,
            Namespace: controlPlaneNamespace,
            Annotations: map[string]string{
                "spectrocloud.com/custom-dns-provided": "",  // ✅ Automatically added
            },
            Labels: map[string]string{
                "hypershift.openshift.io/cluster": hcluster.Name,
                "platform":                        "maas",
            },
        },
    }
    
    // ... rest of implementation ...
}
```

### **Why This Annotation is Required**

The `spectrocloud.com/custom-dns-provided=""` annotation tells the `capi-provider-maas` that:
1. **Custom DNS is already configured** for this cluster
2. **No additional DNS setup is needed** by the provider
3. **The provider should skip DNS configuration** steps

Without this annotation, the provider may attempt to configure DNS services that conflict with existing infrastructure.

## **NEW: Resource Management and Sync Period Configuration**

The MaaS CAPI provider deployment now includes proper resource management and sync period configuration:

### **Resource Limits and Requests**

```go
func (p *MaaS) CAPIProviderDeploymentSpec(hcluster *hyperv1.HostedCluster, _ *hyperv1.HostedControlPlane) (*appsv1.DeploymentSpec, error) {
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
                            "--sync-period=15m",  // ✅ Configurable sync period
                        },
                        Resources: corev1.ResourceRequirements{  // ✅ Resource management
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
```

### **Resource Configuration Details**

- **CPU Limits**: 200m (0.2 CPU cores)
- **Memory Limits**: 100Mi (100 MiB)
- **CPU Requests**: 200m (0.2 CPU cores)  
- **Memory Requests**: 20Mi (20 MiB)
- **Sync Period**: 15 minutes (configurable via `--sync-period=15m`)

### **Benefits of Resource Management**

1. **Predictable Performance**: Resource limits prevent the controller from consuming excessive resources
2. **Cluster Stability**: Resource requests ensure the controller gets the resources it needs
3. **Cost Control**: Prevents resource over-allocation in cloud environments
4. **Monitoring**: Resource usage can be properly tracked and alerted on

### **Benefits of Sync Period Configuration**

1. **Performance Tuning**: 15-minute sync period balances responsiveness with resource usage
2. **Customizable**: Can be adjusted based on cluster size and requirements
3. **Consistent Behavior**: Follows the same pattern as other CAPI providers

## **Summary of Full MaaS CAPI Integration**

We have successfully implemented **full MaaS CAPI integration** in HyperShift, which includes:

✅ **Scheme Registration**: MaaS CAPI types are registered in the operator's scheme  
✅ **CRD Assets**: MaaSCluster CRD is embedded in the HyperShift binary  
✅ **Managed Resources**: MaaSCluster resources are tracked by the operator  
✅ **Type Safety**: Platform implementation uses typed objects instead of unstructured  
✅ **Consistency**: Follows the same pattern as AWS, Azure, and OpenStack platforms  
✅ **Automatic Annotations**: Required `spectrocloud.com/custom-dns-provided=""` annotation is automatically added  
✅ **Resource Management**: Proper CPU and memory limits/requests for the CAPI provider  
✅ **Sync Period Configuration**: Configurable `--sync-period=15m` argument  

### **Files Modified/Added**

1. **`support/api/scheme.go`** - Added MaaS CAPI scheme registration
2. **`support/util/resources.go`** - Added MaaS managed resources
3. **`