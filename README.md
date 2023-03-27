## 部署 ecnet edge 集群

```bash
LOCAL_REGISTRY="${LOCAL_REGISTRY:-localhost:5000}"

export ecnet_namespace=ecnet-system
export ecnet_mesh_name=ecnet

ecnet install \
    --mesh-name "$ecnet_mesh_name" \
    --ecnet-namespace "$ecnet_namespace" \
    --set=ecnet.image.registry=${LOCAL_REGISTRY}/flomesh \
    --set=ecnet.image.tag=latest \
    --set=ecnet.image.pullPolicy=Always \
    --set=ecnet.sidecarLogLevel=error \
    --set=ecnet.controllerLogLevel=warn \
    --timeout=900s
```

## 部署模拟客户端

```bash
kubectl create namespace demo
cat <<EOF | kubectl apply -n demo -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sleep
---
apiVersion: v1
kind: Service
metadata:
  name: sleep
  labels:
    app: sleep
    service: sleep
spec:
  ports:
  - port: 80
    name: http
  selector:
    app: sleep
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sleep
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sleep
  template:
    metadata:
      labels:
        app: sleep
    spec:
      terminationGracePeriodSeconds: 0
      serviceAccountName: sleep
      containers:
      - name: sleep
        image: local.registry/ubuntu:20.04
        imagePullPolicy: Always
        command: ["/bin/sleep", "infinity"]
      nodeName: node2
EOF

kubectl wait --for=condition=ready pod -n demo -l app=sleep --timeout=180s
```

## 模拟导入多集群服务

```bash
kubectl create namespace pipy

cat <<EOF | kubectl apply -f -
apiVersion: flomesh.io/v1alpha1
kind: ServiceImport
metadata:
  name: pipy-ok
  namespace: pipy
spec:
  ports:
  - endpoints:
    - clusterKey: default/default/default/cluster3
      target:
        host: 3.226.203.163
        ip: 3.226.203.163
        path: /
        port: 80
    port: 8080
    protocol: TCP
  serviceAccountName: '*'
  type: ClusterSetIP
EOF

cat <<EOF | kubectl apply -f -
apiVersion: flomesh.io/v1alpha1
kind: GlobalTrafficPolicy
metadata:
  namespace: pipy
  name: pipy-ok
spec:
  lbType: ActiveActive
EOF
```

## 测试

测试指令:

```bash
sleep_client="$(kubectl get pod -n demo -l app=sleep -o jsonpath='{.items[0].metadata.name}')"
kubectl exec ${sleep_client} -n demo -- curl -sI pipy-ok.pipy:8080
```

期望结果:

```bash
HTTP/1.1 200 OK
date: Sat, 25 Mar 2023 16:47:17 GMT
content-type: text/html; charset=utf-8
content-length: 9593
server: gunicorn/19.9.0
access-control-allow-origin: *
access-control-allow-credentials: true
connection: keep-alive
```

