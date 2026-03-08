# Leader Election for Kubernetes

## Goal
Add leader election to worker so only one instance processes jobs when running in Kubernetes.

## Progress

1. **Kubernetes Detection** - Added check for `KUBERNETES_SERVICE_HOST` env var and logging
2. **Leader Election Implementation** - Added lease-based leader election with logging for leader status changes
3. **RBAC Setup** - Created ServiceAccount, Role, and RoleBinding for lease permissions in `k8s/rbac.yaml`
4. **Deployment Updates** - Added serviceAccountName and POD_NAME/POD_NAMESPACE env vars to worker deployment
