# Running HyperShift on Kubernetes Management Clusters

This document provides instructions for running HyperShift on regular Kubernetes clusters (non-OpenShift) by installing the required OpenShift configuration CRDs.

## Problem

HyperShift was originally designed to run on OpenShift management clusters and expects certain OpenShift configuration CRDs to exist. When running on regular Kubernetes clusters, the control-plane-operator fails with errors like:

```
"no matches for kind \"Infrastructure\" in version \"config.openshift.io/v1\""
```

## Solution

Install the missing OpenShift configuration CRDs manually on your Kubernetes management cluster.

## Prerequisites

- Access to a Kubernetes management cluster with cluster-admin permissions
- kubectl configured to access the management cluster
- HyperShift operator already installed

## Step-by-Step Installation

### 1. Create the Infrastructure CRD

Create the OpenShift Infrastructure CRD that HyperShift expects:

```bash
kubectl apply -f - <<'EOF'
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: infrastructures.config.openshift.io
  annotations:
    release.openshift.io/bootstrap-required: "true"
spec:
  group: config.openshift.io
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          apiVersion:
            type: string
          kind:
            type: string
          metadata:
            type: object
          spec:
            type: object
            properties:
              cloudConfig:
                type: object
                properties:
                  key:
                    type: string
                  name:
                    type: string
              platformSpec:
                type: object
                properties:
                  type:
                    type: string
                    enum: ["", "AWS", "Azure", "BareMetal", "GCP", "Libvirt", "OpenStack", "None", "VSphere", "oVirt", "KubeVirt", "EquinixMetal", "PowerVS", "AlibabaCloud", "Nutanix", "External"]
                  aws:
                    type: object
                  azure:
                    type: object
                  gcp:
                    type: object
                  openstack:
                    type: object
                  ovirt:
                    type: object
                  vsphere:
                    type: object
                  baremetal:
                    type: object
                  none:
                    type: object
          status:
            type: object
            properties:
              apiServerURL:
                type: string
              apiServerInternalURL:
                type: string
              etcdDiscoveryDomain:
                type: string
              infrastructureName:
                type: string
              platform:
                type: string
                enum: ["", "AWS", "Azure", "BareMetal", "GCP", "Libvirt", "OpenStack", "None", "VSphere", "oVirt", "KubeVirt", "EquinixMetal", "PowerVS", "AlibabaCloud", "Nutanix", "External"]
              platformStatus:
                type: object
                properties:
                  type:
                    type: string
                    enum: ["", "AWS", "Azure", "BareMetal", "GCP", "Libvirt", "OpenStack", "None", "VSphere", "oVirt", "KubeVirt", "EquinixMetal", "PowerVS", "AlibabaCloud", "Nutanix", "External"]
                  aws:
                    type: object
                  azure:
                    type: object
                  gcp:
                    type: object
                  openstack:
                    type: object
                  ovirt:
                    type: object
                  vsphere:
                    type: object
                  baremetal:
                    type: object
  scope: Cluster
  names:
    plural: infrastructures
    singular: infrastructure
    kind: Infrastructure
    listKind: InfrastructureList
EOF
```

### 2. Wait for CRD to be Established

```bash
kubectl wait --for condition=established --timeout=60s crd/infrastructures.config.openshift.io
```

### 3. Create the Infrastructure Resource

Create the Infrastructure resource that describes your management cluster. **Important**: Replace the `apiServerURL` with your actual management cluster's external API server endpoint.

```bash
# Replace YOUR_MANAGEMENT_CLUSTER_API_ENDPOINT with your actual endpoint
kubectl apply -f - <<EOF
apiVersion: config.openshift.io/v1
kind: Infrastructure
metadata:
  name: cluster
spec:
  platformSpec:
    type: BareMetal  # Use appropriate platform type for your infrastructure
status:
  apiServerURL: "https://YOUR_MANAGEMENT_CLUSTER_API_ENDPOINT:6443"
  apiServerInternalURL: "https://kubernetes.default.svc:443"
  infrastructureName: "hypershift-management"
  platform: BareMetal  # Use appropriate platform type for your infrastructure
  platformStatus:
    type: BareMetal  # Use appropriate platform type for your infrastructure
EOF
```

### 4. Create RBAC Permissions

The HyperShift control-plane-operator needs permissions to read Infrastructure resources:

```bash
# Create ClusterRole for reading Infrastructure resources
kubectl create clusterrole infrastructure-reader \
  --verb=get,list,watch \
  --resource=infrastructures.config.openshift.io

# Bind the role to the control-plane-operator ServiceAccount
# Replace HOSTED_CLUSTER_NAMESPACE with your actual hosted cluster namespace
kubectl create clusterrolebinding control-plane-operator-infrastructure \
  --clusterrole=infrastructure-reader \
  --serviceaccount=HOSTED_CLUSTER_NAMESPACE:control-plane-operator
```

### 5. Restart Control Plane Operator

Restart the control-plane-operator to pick up the new permissions:

```bash
# Replace HOSTED_CLUSTER_NAMESPACE with your actual hosted cluster namespace
kubectl rollout restart deployment/control-plane-operator -n HOSTED_CLUSTER_NAMESPACE

# Wait for rollout to complete
kubectl rollout status deployment/control-plane-operator -n HOSTED_CLUSTER_NAMESPACE --timeout=60s
```

## Configuration Options

### Platform Types

Choose the appropriate platform type for your infrastructure:

- `BareMetal`: For MAAS, bare metal, or on-premises infrastructure
- `AWS`: For AWS-based management clusters
- `Azure`: For Azure-based management clusters
- `GCP`: For Google Cloud-based management clusters
- `VSphere`: For VMware vSphere infrastructure
- `None`: For generic/unspecified platforms

### API Server URL Discovery

To find your management cluster's external API server endpoint:

```bash
# Method 1: Check current kubectl context
kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}'

# Method 2: Check kubernetes service (if using LoadBalancer)
kubectl get svc kubernetes -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

# Method 3: Check nodes external IPs (for NodePort services)
kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="ExternalIP")].address}'
```

## Verification

### 1. Verify CRD Installation

```bash
kubectl get crd infrastructures.config.openshift.io
```

### 2. Verify Infrastructure Resource

```bash
kubectl get infrastructure cluster -o yaml
```

### 3. Check Control Plane Operator Logs

```bash
# Replace HOSTED_CLUSTER_NAMESPACE with your actual hosted cluster namespace
kubectl logs -n HOSTED_CLUSTER_NAMESPACE deployment/control-plane-operator --tail=20
```

You should no longer see errors about missing Infrastructure CRDs.

## Troubleshooting

### Permission Errors

If you see permission errors like:
```
infrastructures.config.openshift.io is forbidden: User "system:serviceaccount:..." cannot list resource "infrastructures"
```

Ensure you've created the RBAC permissions in step 4 and restarted the control-plane-operator.

### CRD Not Established

If the CRD doesn't become established:
```bash
kubectl describe crd infrastructures.config.openshift.io
```

Check for any validation errors in the CRD definition.

### Wrong API Server URL

If the Infrastructure resource has the wrong API server URL, update it:
```bash
kubectl patch infrastructure cluster --type='merge' -p='{"status":{"apiServerURL":"https://YOUR_CORRECT_ENDPOINT:6443"}}'
```

## Alternative Solutions

1. **Use OpenShift Management Cluster**: Run HyperShift on an actual OpenShift cluster (recommended for production)
2. **Custom HyperShift Build**: Modify HyperShift to handle missing OpenShift CRDs gracefully
3. **Install Full OpenShift Config Operator**: Install the complete OpenShift configuration operator (more complex)

## Notes

- This is a workaround for running HyperShift on Kubernetes clusters
- For production deployments, consider using an OpenShift management cluster
- The Infrastructure resource provides platform information that HyperShift uses for configuration decisions
- Additional OpenShift CRDs may be needed depending on your HyperShift configuration and platform requirements
