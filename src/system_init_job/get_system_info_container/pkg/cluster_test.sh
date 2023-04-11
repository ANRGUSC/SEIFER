# Delete existing chaos mesh
kubectl delete workflows --all
# Create new workflows
kubectl create -f /root/configs/ --recursive