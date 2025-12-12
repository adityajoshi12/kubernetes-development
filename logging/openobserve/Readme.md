# Install OpenObserve in K8s Cluster

## Prerequisites

- Kubernetes cluster running (1.19+)
- kubectl configured and connected to your cluster
- Helm 3.x installed (optional, for alternative installation methods)
- Minimum 2GB RAM and 2 CPU cores available

## Installation Steps

### 1. Create Namespace

Create a dedicated namespace for OpenObserve:

```bash
kubectl create namespace openobserve
```

### 2. Deploy OpenObserve

Deploy OpenObserve using the official StatefulSet manifest:

```bash
kubectl apply -f https://raw.githubusercontent.com/zinclabs/openobserve/main/deploy/k8s/statefulset.yaml
```

Wait for the pods to be ready:

```bash
kubectl wait --for=condition=ready pod -l app=openobserve -n openobserve --timeout=300s
```

### 3. Access OpenObserve

Port-forward to access the UI locally:

```bash
kubectl -n openobserve port-forward svc/openobserve 5080:5080
```

Open http://localhost:5080 in your browser.

## Default Credentials

The default login credentials are:

- **Username:** `root@example.com`
- **Password:** `Complexpass#123`

> ⚠️ **Security Warning:** Change these credentials immediately after first login.

## Configure Fluent Bit for Log Collection

### 1. Create Namespace for Logging

Create a dedicated namespace for logging infrastructure:

```bash
kubectl create namespace kube-logging
```

### 2. Create Authentication Secret

Store OpenObserve credentials securely for Fluent Bit authentication:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: fluent-bit-http-auth
  namespace: kube-logging
type: Opaque
stringData:
  OPENOBSERVE_USER: "root@example.com"
  OPENOBSERVE_PASS: "Complexpass#123"
EOF
```

> 💡 **Tip:** Update these credentials if you changed the OpenObserve defaults.

### 3. Create Fluent Bit Configuration

Configure Fluent Bit to collect container logs and enrich them with Kubernetes metadata:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
  namespace: kube-logging
data:
  fluent-bit.conf: |
    [SERVICE]
        Flush        1
        Log_Level    info
        Parsers_File parsers.conf

    # Collect container logs from Kubernetes nodes
    [INPUT]
        Name tail
        Path /var/log/containers/*.log
        
        # Exclude system and logging namespace logs to prevent loops
        Exclude_Path /var/log/containers/fluent-bit-*.log,/var/log/containers/*_kube-logging_*.log,/var/log/containers/*_openobserve_*.log
        
        Parser docker
        Tag kube.*
        Refresh_Interval 5
        Mem_Buf_Limit 5MB
        Skip_Long_Lines On

    # Enrich logs with Kubernetes metadata
    # Adds: pod_name, namespace_name, container_name, labels, annotations
    [FILTER]
        Name                kubernetes
        Match               kube.*
        Kube_URL            https://kubernetes.default.svc:443
        Kube_CA_File        /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        Kube_Token_File     /var/run/secrets/kubernetes.io/serviceaccount/token
        Merge_Log           On
        Keep_Log            Off
        Merge_Log_Key       log
        K8S-Logging.Parser  On
        K8S-Logging.Exclude On
        Labels              On
        Annotations         Off

    # Flatten nested Kubernetes fields for easier querying
    [FILTER]
        Name   nest
        Match  kube.*
        Operation lift
        Nested_under kubernetes

    # Send logs to OpenObserve
    [OUTPUT]
        Name http
        Match *
        URI /api/default/default/_json
        Host openobserve-lb.openobserve
        Port 5080
        tls Off
        Format json
        Json_date_key _timestamp
        Json_date_format iso8601
        
        # Authentication from environment variables
        HTTP_User  \${OPENOBSERVE_USER}
        HTTP_Passwd \${OPENOBSERVE_PASS}
        
        # Enable compression for bandwidth efficiency
        compress gzip
        
        # Retry configuration
        Retry_Limit 3

  parsers.conf: |
    [PARSER]
        Name   docker
        Format json
        Time_Key time
        Time_Format %Y-%m-%dT%H:%M:%S.%L
        Time_Keep On
EOF
```

### 4. Create RBAC Resources

Grant Fluent Bit the necessary permissions to read pod and namespace metadata:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: fluent-bit
  namespace: kube-logging
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: fluent-bit-read
rules:
- apiGroups: [""]
  resources: ["pods", "namespaces"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: fluent-bit-read
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: fluent-bit-read
subjects:
- kind: ServiceAccount
  name: fluent-bit
  namespace: kube-logging
EOF
```

### 5. Deploy Fluent Bit DaemonSet

Deploy Fluent Bit on every node to collect logs cluster-wide:

```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluent-bit
  namespace: kube-logging
  labels:
    app: fluent-bit
spec:
  selector:
    matchLabels:
      app: fluent-bit
  template:
    metadata:
      labels:
        app: fluent-bit
    spec:
      serviceAccountName: fluent-bit
      tolerations:
      # Allow running on all nodes including master nodes
      - effect: NoSchedule
        operator: Exists
      containers:
      - name: fluent-bit
        image: docker.io/fluent/fluent-bit:3.0
        resources:
          limits:
            memory: 200Mi
          requests:
            cpu: 100m
            memory: 100Mi
        envFrom:
        - secretRef:
            name: fluent-bit-http-auth
        volumeMounts:
        - name: config
          mountPath: /fluent-bit/etc/
        - name: varlog
          mountPath: /var/log
          readOnly: true
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
      volumes:
      - name: config
        configMap:
          name: fluent-bit-config
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibdockercontainers
        hostPath:
          path: /var/lib/docker/containers
EOF
```

### 6. Verify Fluent Bit Deployment

Check that Fluent Bit pods are running on all nodes:

```bash
kubectl get pods -n kube-logging -o wide
kubectl logs -n kube-logging -l app=fluent-bit --tail=50
```

Expected output: One pod per node in `Running` state.

## Deploying Sample Application

### Build and Load Container Image

Build the sample logging application and load it into your cluster:

```bash
# Build the Docker image
docker build -t logging-app:1.0 .

# Load image into kind cluster (adjust for other cluster types)
kind load docker-image logging-app:1.0

# Deploy the application
kubectl apply -f deployment.yaml
```

### Verify Application Deployment

```bash
kubectl get pods -n logging-app
kubectl wait --for=condition=ready pod -l app=logging-app -n logging-app --timeout=60s
```

### Generate Test Logs

Generate various log entries to test the logging pipeline:

```bash
# Get the pod name
POD=$(kubectl get pod -n logging-app -l app=logging-app -o jsonpath='{.items[0].metadata.name}')

# Generate different log types
kubectl exec -n logging-app $POD -- wget -qO- http://localhost:8080/
kubectl exec -n logging-app $POD -- wget -qO- http://localhost:8080/health
kubectl exec -n logging-app $POD -- wget -qO- http://localhost:8080/api/users
kubectl exec -n logging-app $POD -- wget -qO- http://localhost:8080/api/process
kubectl exec -n logging-app $POD -- wget -qO- http://localhost:8080/api/error
```

### View Logs in OpenObserve

1. Navigate to http://localhost:5080
2. Log in with your credentials
3. Go to the **Logs** section
4. Filter by `namespace_name:logging-app` to see your application logs
5. Explore the Kubernetes metadata fields added by Fluent Bit

## Troubleshooting

### Fluent Bit Not Sending Logs

Check Fluent Bit logs for errors:

```bash
kubectl logs -n kube-logging -l app=fluent-bit --tail=100
```

### OpenObserve Not Accessible

Verify OpenObserve is running:

```bash
kubectl get pods -n openobserve
kubectl logs -n openobserve -l app=openobserve
```

### No Logs Appearing in OpenObserve

1. Verify network connectivity between Fluent Bit and OpenObserve
2. Check authentication credentials in the secret
3. Ensure the service name `openobserve-lb.openobserve` resolves correctly

## Cleanup

Remove all installed resources:

```bash
kubectl delete namespace logging-app
kubectl delete namespace kube-logging
kubectl delete namespace openobserve
kubectl delete clusterrole fluent-bit-read
kubectl delete clusterrolebinding fluent-bit-read
```