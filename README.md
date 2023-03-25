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

## 部署本地业务服务

```
kubectl create namespace pipy
kubectl apply -n pipy -f https://raw.githubusercontent.com/cybwan/ecnet-edge-start-demo/main/demo/multi-cluster/pipy-ok-c1.pipy.yaml

#等待依赖的 POD 正常启动
sleep 3
kubectl wait --for=condition=ready pod -n pipy -l app=pipy-ok-c1 --timeout=180s
```

## 模拟导入多集群服务

```
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
        host: 192.168.127.91
        ip: 192.168.127.91
        path: /c3/ok
        port: 8093
    - clusterKey: default/default/default/cluster1
      target:
        host: 192.168.127.91
        ip: 192.168.127.91
        path: /c1/ok
        port: 8091
    name: pipy
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

