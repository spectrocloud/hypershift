# MAAS Platform Integration for HyperShift

This document describes the changes made to enable MAAS (Metal as a Service) platform support in HyperShift.

## Overview

MAAS platform support allows HyperShift to provision and manage OpenShift clusters on MAAS infrastructure. This includes:
- HostedCluster creation with MAAS platform configuration
- NodePool management for MAAS worker nodes
- CAPI provider integration for machine provisioning
- Proper service publishing with NodePort strategy

## Key Changes Made

### 1. API Types and Constants

**File**: `api/hypershift/v1beta1/hostedcluster_types.go`
- Added `MAASPlatform` constant
- Added `NodePortPublishingStrategy` for MAAS service exposure

### 2. MAAS HostedCluster Platform Logic

**File**: `hypershift-operator/controllers/hostedcluster/internal/platform/maas/maas.go`

#### Key Functions:
- `ReconcileCredentials`: Creates MAAS credentials secret in control plane namespace
- `CAPIProviderDeploymentSpec`: Deploys the MAAS CAPI provider
- `ReconcileCAPIProvider`: Main reconciliation logic for MAAS platform

#### Critical Fix:
```go
// Fixed secret name mismatch - use IdentityRef.Name instead of generated name
controlPlaneSecret := &corev1.Secret{
    ObjectMeta: metav1.ObjectMeta{
        Name:      hcluster.Spec.Platform.MAAS.IdentityRef.Name, // Fixed: was fmt.Sprintf("%s-maas-credentials", hcluster.Name)
        Namespace: controlPlaneNamespace,
        // ...
    },
    // ...
}
```

### 3. MAAS NodePool Platform Logic

**File**: `hypershift-operator/controllers/nodepool/maas/maas.go`

#### Key Functions:
- `ReconcileMachineTemplate`: Creates MaasMachineTemplate with proper image configuration
- `ReconcileMachineSet`: Manages MachineSet for MAAS machines
- `ReconcileMachineDeployment`: Handles MachineDeployment for worker nodes

#### Image Configuration:
- Default image: `ubuntu/focal`
- Can be overridden via `NodePool.Spec.Platform.MAAS.Image`
- Example: `ubuntu-rhcos-with-backup-2`

### 4. CLI Command for MAAS Cluster Creation

**File**: `cmd/cluster/maas/create.go`

#### New Features:
- `--external-api-server-address` flag for NodePort configuration
- Auto-detection of API server address using `core.GetAPIServerAddressByNode`
- Service publishing strategy configuration for NodePort services

#### Example Usage:
```bash
hypershift create cluster maas \
  --name test \
  --namespace clusters \
  --external-api-server-address 10.10.155.171 \
  --base-domain hypershift.spectrocloud.dev \
  --pull-secret pull-secret.json \
  --ssh-key test_id_rsa.pub \
  --maas-server-url https://maas.example.com \
  --maas-api-key <api-key>
```

### 5. Infrastructure Resource Reconciliation

**File**: `control-plane-operator/controllers/hostedcontrolplane/hostedcontrolplane_controller.go`

#### Critical Fix:
Added direct reconciliation of Infrastructure resource for MAAS platforms:

```go
func (r *HostedControlPlaneReconciler) reconcileInfrastructure(ctx context.Context, hcp *hyperv1.HostedControlPlane, createOrUpdate upsert.CreateOrUpdateFN) error {
    // Reconcile Infrastructure resource for MAAS platform
    if hcp.Spec.Platform.Type == hyperv1.MAASPlatform {
        infra := globalconfig.InfrastructureConfig()
        if _, err := createOrUpdate(ctx, r.Client, infra, func() error {
            globalconfig.ReconcileInfrastructure(infra, hcp)
            return nil
        }); err != nil {
            return fmt.Errorf("failed to reconcile infrastructure config: %w", err)
        }
    }
    // ... existing service reconciliation
}
```

**File**: `support/globalconfig/infrastructure.go`

#### Platform Type Configuration:
```go
func ReconcileInfrastructure(infra *configv1.Infrastructure, hcp *hyperv1.HostedControlPlane) {
    platformType := hcp.Spec.Platform.Type
    if platformType == hyperv1.MAASPlatform {
        // MAAS platform should use "None" type in OpenShift Infrastructure resource
        infra.Spec.PlatformSpec.Type = configv1.NonePlatformType
    } else {
        infra.Spec.PlatformSpec.Type = configv1.PlatformType(platformType)
    }
    // ...
}
```

## Configuration Requirements

### 1. MAAS Credentials Secret

Create a secret with MAAS API credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: maas-credentials
  namespace: hypershift
type: Opaque
data:
  maas-server-url: <base64-encoded-url>
  maas-api-key: <base64-encoded-api-key>
```

### 2. HostedCluster Configuration

```yaml
apiVersion: hypershift.openshift.io/v1beta1
kind: HostedCluster
metadata:
  name: test
  namespace: clusters
spec:
  platform:
    type: MAAS
    maas:
      identityRef:
        name: maas-credentials
  services:
    - service: APIServer
      servicePublishingStrategy:
        type: NodePort
        nodePort:
          address: 10.10.155.171  # External API server address
  dns:
    baseDomain: hypershift.spectrocloud.dev
```

### 3. NodePool Configuration

```yaml
apiVersion: hypershift.openshift.io/v1beta1
kind: NodePool
metadata:
  name: test
  namespace: clusters
spec:
  clusterName: test
  platform:
    type: MAAS
    maas:
      identityRef:
        name: maas-credentials
      image: ubuntu-rhcos-with-backup-2  # Custom MAAS image
      zone: az1
  management:
    upgradeType: Replace
    replace:
      strategy: RollingUpdate
      rollingUpdate:
        maxSurge: 1
        maxUnavailable: 0
```

## Troubleshooting

### Common Issues and Solutions

1. **CAPI Provider Pod Fails to Start**
   - **Error**: `secret "maas-credentials" not found`
   - **Solution**: Ensure secret name matches `IdentityRef.Name` in HostedCluster spec

2. **HostedCluster Services Immutable Error**
   - **Error**: `HostedCluster services are immutable, cannot patch nodePort.address`
   - **Solution**: Delete and recreate HostedCluster with correct configuration

3. **Cluster Network Operator Stuck**
   - **Error**: `Infrastructure.config.openshift.io "cluster" is invalid: spec.platformSpec.type: Unsupported value: "MAAS"`
   - **Solution**: Ensure Infrastructure resource is reconciled with `platformSpec.type: "None"`

4. **Machine Stuck in Pending State**
   - **Cause**: MAAS deployment failure or machine not joining cluster
   - **Solution**: Check MAAS logs, verify image availability, check network connectivity

5. **CAPI Provider Reconciliation Loop**
   - **Cause**: Machine stuck in "Deploying" state in MAAS
   - **Solution**: Fix MAAS deployment issues or scale down NodePool to stop reconciliation

### Debug Commands

```bash
# Check HostedCluster status
kubectl get hostedcluster -n clusters

# Check NodePool status
kubectl get nodepool -n clusters

# Check machine status
kubectl get machine -n clusters-<cluster-name>

# Check CAPI provider logs
kubectl logs -n clusters-<cluster-name> deployment/capi-provider

# Check Infrastructure resource
kubectl get infrastructure cluster -o yaml

# Check for pending CSRs
kubectl get csr
```

## Testing

### 1. Create Test Cluster

```bash
# Set kubeconfig
export KUBECONFIG=guy-hypershift.kubeconfig

# Create MAAS HostedCluster
hypershift create cluster maas \
  --name test \
  --namespace clusters \
  --external-api-server-address 10.10.155.171 \
  --base-domain hypershift.spectrocloud.dev \
  --pull-secret pull-secret.json \
  --ssh-key test_id_rsa.pub \
  --maas-server-url https://maas.example.com \
  --maas-api-key <api-key>
```

### 2. Create NodePool

```bash
# Create NodePool with custom image
kubectl apply -f - <<EOF
apiVersion: hypershift.openshift.io/v1beta1
kind: NodePool
metadata:
  name: test
  namespace: clusters
spec:
  clusterName: test
  platform:
    type: MAAS
    maas:
      identityRef:
        name: maas-credentials
      image: ubuntu-rhcos-with-backup-2
      zone: az1
  management:
    upgradeType: Replace
    replace:
      strategy: RollingUpdate
      rollingUpdate:
        maxSurge: 1
        maxUnavailable: 0
EOF
```

### 3. Monitor Deployment

```bash
# Watch HostedCluster progress
kubectl get hostedcluster test -n clusters -w

# Watch NodePool progress
kubectl get nodepool test -n clusters -w

# Watch machine creation
kubectl get machine -n clusters-test -w
```

## Architecture

The MAAS platform integration follows this flow:

1. **HostedCluster Creation**: Creates MAAS platform configuration and deploys CAPI provider
2. **CAPI Provider Deployment**: Manages MAAS machine lifecycle through Cluster API
3. **NodePool Management**: Creates MachineSet and MachineDeployment for worker nodes
4. **Machine Provisioning**: MAAS provisions physical machines with specified images
5. **Node Registration**: Machines join the Kubernetes cluster and become worker nodes

## Dependencies

- MAAS server with API access
- MAAS images configured (e.g., `ubuntu-rhcos-with-backup-2`)
- Network connectivity between management cluster and MAAS
- Proper DNS configuration for base domain
- Valid pull secrets and SSH keys

## Future Improvements

- Add timeout mechanisms for stuck deployments
- Implement better error handling and retry logic
- Add support for MAAS-specific machine configurations
- Improve monitoring and observability
- Add support for MAAS resource pools and constraints
