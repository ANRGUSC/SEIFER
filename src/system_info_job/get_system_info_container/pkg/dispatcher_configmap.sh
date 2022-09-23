kubectl create configmap dispatcher-config --from-file=/config/node_info.json
kubectl label configmap dispatcher-config app=dispatcher