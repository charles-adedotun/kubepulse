# KubePulse Kubernetes Deployment

This directory contains the Kubernetes manifests for deploying KubePulse to your cluster.

## Prerequisites

- Kubernetes cluster version 1.19+
- kubectl configured to access your cluster
- (Optional) cert-manager for automatic TLS certificates
- (Optional) metrics-server for HPA functionality
- (Optional) Ingress controller (e.g., nginx-ingress) for external access

## Quick Start

1. **Create the namespace and RBAC resources:**
   ```bash
   kubectl apply -f rbac.yaml
   ```

2. **Create secrets (optional, for AI features):**
   ```bash
   cp secrets-template.yaml secrets.yaml
   # Edit secrets.yaml with your values
   kubectl apply -f secrets.yaml
   ```

3. **Deploy KubePulse:**
   ```bash
   kubectl apply -f deployment.yaml
   ```

4. **Verify the deployment:**
   ```bash
   kubectl -n kubepulse-system get pods
   kubectl -n kubepulse-system logs -l app.kubernetes.io/name=kubepulse
   ```

## Configuration

### Basic Configuration

The main configuration is stored in the `kubepulse-config` ConfigMap. Key settings include:

- **Monitoring interval**: How often health checks run (default: 30s)
- **Enabled checks**: Which health checks to run
- **AI features**: Enable/disable AI analysis and remediation
- **Alert channels**: Where to send alerts

### Security Configuration

KubePulse follows security best practices:

- **Read-only by default**: Only requests read permissions to cluster resources
- **Non-root user**: Runs as UID 1000
- **Read-only root filesystem**: Prevents runtime modifications
- **Network policies**: Restricts ingress/egress traffic
- **Pod Security Policy**: Enforces security constraints

### AI Features

To enable AI features:

1. Obtain a Claude API key
2. Add it to the secrets:
   ```bash
   kubectl create secret generic kubepulse-secrets \
     --from-literal=claude-api-key=YOUR_API_KEY \
     -n kubepulse-system
   ```
3. Set `ai.enabled: true` in the ConfigMap

### Remediation Mode

⚠️ **WARNING**: Remediation mode grants write permissions to your cluster. Enable only after careful review.

To enable remediation:

1. Apply the remediation ClusterRole:
   ```bash
   kubectl apply -f - <<EOF
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRoleBinding
   metadata:
     name: kubepulse-remediation
   roleRef:
     apiGroup: rbac.authorization.k8s.io
     kind: ClusterRole
     name: kubepulse-remediation
   subjects:
     - kind: ServiceAccount
       name: kubepulse
       namespace: kubepulse-system
   EOF
   ```

2. Set `ai.enableRemediation: true` in the ConfigMap

3. Restart the KubePulse pod

## Accessing KubePulse

### Port Forward (Development)

```bash
kubectl -n kubepulse-system port-forward svc/kubepulse 8080:8080
```

Access at http://localhost:8080

### NodePort (Testing)

The NodePort service exposes KubePulse on port 30080:
```bash
http://<node-ip>:30080
```

### Ingress (Production)

1. Update the Ingress resource with your domain
2. Ensure you have an Ingress controller installed
3. Apply the deployment
4. Access at https://kubepulse.example.com

## Monitoring

### Prometheus Metrics

KubePulse exposes Prometheus metrics on port 9090:

```bash
kubectl -n kubepulse-system port-forward svc/kubepulse 9090:9090
curl http://localhost:9090/metrics
```

### Health Checks

- Liveness: `/health/live`
- Readiness: `/health/ready`

## Storage

KubePulse uses a PersistentVolumeClaim for storing:
- AI analysis history
- Baseline metrics
- Learned patterns

The default size is 10Gi. Adjust based on your cluster size and retention needs.

## Troubleshooting

### Pod Fails to Start

Check logs:
```bash
kubectl -n kubepulse-system logs -l app.kubernetes.io/name=kubepulse
```

Common issues:
- Missing secrets (if AI is enabled)
- Insufficient permissions
- PVC not bound

### No Data Showing

1. Check if health checks are running:
   ```bash
   kubectl -n kubepulse-system logs -l app.kubernetes.io/name=kubepulse | grep "Running health check"
   ```

2. Verify RBAC permissions:
   ```bash
   kubectl auth can-i get pods --as=system:serviceaccount:kubepulse-system:kubepulse
   ```

### AI Features Not Working

1. Verify Claude API key is set:
   ```bash
   kubectl -n kubepulse-system get secret kubepulse-secrets
   ```

2. Check AI client initialization:
   ```bash
   kubectl -n kubepulse-system logs -l app.kubernetes.io/name=kubepulse | grep "AI"
   ```

## Customization

### Using Custom Images

Build and push your image:
```bash
docker build -t myregistry/kubepulse:custom .
docker push myregistry/kubepulse:custom
```

Update the deployment:
```bash
kubectl -n kubepulse-system set image deployment/kubepulse kubepulse=myregistry/kubepulse:custom
```

### Adjusting Resource Limits

Edit the deployment to modify resource requests/limits based on your cluster size:

- Small cluster (<50 nodes): 100m CPU, 256Mi memory
- Medium cluster (50-200 nodes): 500m CPU, 512Mi memory  
- Large cluster (>200 nodes): 1 CPU, 1Gi memory

### Multi-Cluster Deployment

For monitoring multiple clusters:

1. Deploy KubePulse to each cluster
2. Use unique namespaces or labels
3. Configure external database for centralized storage
4. Set up federation for cross-cluster views

## Security Considerations

1. **Secrets Management**: Use a secrets management solution (e.g., Sealed Secrets, HashiCorp Vault)
2. **Network Policies**: Adjust the NetworkPolicy based on your cluster setup
3. **RBAC**: Review and minimize permissions based on enabled features
4. **Pod Security**: Ensure PodSecurityPolicy or Pod Security Standards are enforced
5. **Image Scanning**: Scan KubePulse images for vulnerabilities before deployment

## Maintenance

### Backup

Backup the PersistentVolume regularly:
```bash
kubectl -n kubepulse-system exec -it kubepulse-0 -- tar czf /tmp/backup.tar.gz /data
kubectl -n kubepulse-system cp kubepulse-0:/tmp/backup.tar.gz ./backup.tar.gz
```

### Updates

1. Check for new versions
2. Review changelog for breaking changes
3. Test in non-production environment
4. Apply updates:
   ```bash
   kubectl -n kubepulse-system set image deployment/kubepulse kubepulse=kubepulse/kubepulse:new-version
   ```

### Cleanup

To remove KubePulse:
```bash
kubectl delete -f deployment.yaml
kubectl delete -f rbac.yaml
kubectl delete namespace kubepulse-system
```

## Support

- Documentation: https://github.com/kubepulse/kubepulse/docs
- Issues: https://github.com/kubepulse/kubepulse/issues
- Community: https://kubepulse.io/community