# Start the cluster from here, run this script on any node in the cluster
kubectl delete statefulset nfs-provisioner
kubectl delete deployments,jobs --all
kubectl delete configmap dispatcher-config
# Make sure that nfs directory exists
#minikube ssh -n 5-node-cluster "sudo mkdir /var/nfs_backing_pv"
#num_nodes=9
#cluster_name="9-node-cluster"
#minikube ssh -n $cluster_name "sudo mkdir /var/nfs_backing_pv"
#for ((i=2; i <= num_nodes; i++))
#do
#  suffix="m$i"
#  if [ $i -lt 10 ]
#  then
#    suffix="m0$i"
#  fi
#  minikube ssh -n "$cluster_name-$suffix" "sudo mkdir /var/nfs_backing_pv"
#done

kubectl delete pvc nfs-backing-pvc
kubectl delete pvc nfs
kubectl delete pv --all
kubectl delete services -l app=defer
kubectl delete services -l app=nfs-provisioner
sh /Users/arjun/Desktop/GitHub/SEIFER/src/nfs/config_nfs_support.sh
kubectl apply -f /Users/arjun/Desktop/GitHub/SEIFER/src/imagepullsecrets.yaml
kubectl apply -f /Users/arjun/Desktop/GitHub/SEIFER/src/cluster_rbac.yaml

# Init job
kubectl apply -f /Users/arjun/Desktop/GitHub/SEIFER/src/system_init_job.yaml