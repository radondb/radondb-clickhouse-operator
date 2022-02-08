# Task: get access to Kubernetes cluster 

## Access k8s cluster on AWS

### Create new context
```bash
kubectl config set-context radondb.k8s.local
```

### Access k8s service locally
We can access any service inside k8s cluster on `localhost` via port-forwarding feature.
Example: forwarding Graphana dashboard to `localhost:3090`
```bash
kubectl --namespace=grafana port-forward service/grafana-service 3090:3000
```
Point browser to `http://localhost:3090` in order to access Grafana  

### Update k8s config
Add new `cluster` section to kubernetes config. It is usually located at `~/.kube/config`
```txt
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: # AWS key
    server: # k8s API server URL like https://k8s.api.server.URL.on.Amazon.goes.here.amazonaws.com
  name: radondb.k8s.local
contexts:
- context:
    cluster: radondb.k8s.local
    user: radondb.k8s.local
  name: radondb.k8s.local
current-context: radondb.k8s.local
kind: Config
preferences: {}
users:
- name: radondb.k8s.local
  user:
    client-certificate-data: # AWS key
    client-key-data: # AWS key
    password: # password here
    username: admin
- name: radondb.k8s.local-basic-auth
  user:
    password: # password here
    username: admin
```
