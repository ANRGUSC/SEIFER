kubectl apply -f ./nfs/nfs_service.yaml
kubectl apply -f ./nfs/nfs_rbac.yaml
kubectl apply -f ./nfs/nfs_storageclass.yaml
kubectl apply -f ./nfs/nfs_pvc.yaml
kubectl apply -f ./nfs/nfs_backing_storageclass.yaml
kubectl apply -f ./nfs/nfs_backing_pvc.yaml