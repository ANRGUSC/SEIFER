# Start the cluster from here, run this script on any node in the cluster
# Not deleting configmaps since we don't want to have to rerun system-info-job every time
kubectl delete deployments,jobs --all
#kubectl delete statefulset nfs-provisioner
#sh /Users/arjun/Desktop/GitHub/SEIFER/src/nfs/config_nfs_support.sh
# Make sure that nfs directory exists
#minikube ssh -n minikube "sudo mkdir /var/dispatcher_pv"
#kubectl delete pvc dispatcher-nfs-pvc
#kubectl delete pvc nfs
#kubectl delete pv --all --grace-period=0 --force
#kubectl delete services -l app=defer
kubectl apply -f /Users/arjun/Desktop/GitHub/SEIFER/src/imagepullsecrets.yaml
kubectl apply -f /Users/arjun/Desktop/GitHub/SEIFER/src/cluster_rbac.yaml

# Init job
kubectl apply -f /Users/arjun/Desktop/GitHub/SEIFER/src/system_init_job.yaml