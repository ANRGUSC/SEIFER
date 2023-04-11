package system_info

import (
	"context"
	"fmt"
	"os/exec"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/net"
)

func DispatcherInitJob(ctx context.Context, clientset *kubernetes.Clientset, initNode string) {
	fmt.Println("Initializing dispatcher")
	jobClient := clientset.BatchV1().Jobs(corev1.NamespaceDefault)
	var terminateAfter int32 = 0

	cmd := exec.Command("/bin/sh", "/root/dispatcher_configmap.sh")
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	handle(err)

	// We schedule this job to whichever node we want, since it'll finish before inference starts
	jobName := "dispatcher-init-job"
	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
			Labels: map[string]string{
				"task": "dispatcher",
				"type": "dispatcher",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"task": "dispatcher",
						"type": "dispatcher",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "dispatcher-partitioner",
							Image: "localhost:5000/dispatcher_partitioner:latest",
							// Build image to local container registry and pull
							ImagePullPolicy: corev1.PullAlways,
							VolumeMounts: []corev1.VolumeMount{
								// To place the partitions and dispatcher next node
								{
									Name:      "nfs-volume",
									MountPath: "/nfs",
								},
								// Get node bandwidth/memory info
								{
									Name:      "dispatcher-config",
									MountPath: "/dispatcher_config",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "dispatcher-deploy-pods",
							Image: "localhost:5000/deploy_pods:latest",
							// Build image to local container registry and pull
							ImagePullPolicy: corev1.PullAlways,
							VolumeMounts: []corev1.VolumeMount{
								// To read the partitions directory and find the node names
								{
									Name:      "nfs-volume",
									MountPath: "/nfs",
								},
							},
						},
					},
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "ghcr-imagepull-auth",
						},
					},
					Volumes: []corev1.Volume{
						// Dispatcher Config info (node_info.json)
						{
							Name: "dispatcher-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "dispatcher-config",
									},
								},
							},
						},
						// NFS Server which holds all config data
						{
							Name: "nfs-volume",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "nfs",
								},
							},
						},
						// Communication between Golang sockets and Python processing runtime
						{
							Name: "pipe-communication",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					// To debug partitioner pod
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: "defer-admin-account",
				},
			},
			TTLSecondsAfterFinished: &terminateAfter,
		},
	}

	// Create Dispatcher Init Job
	_, err = jobClient.Create(ctx, jobSpec, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		deletePods := metav1.DeletionPropagation("Background")
		err = jobClient.Delete(ctx, jobName, metav1.DeleteOptions{PropagationPolicy: &deletePods})
		handle(err)
		_, err = jobClient.Create(ctx, jobSpec, metav1.CreateOptions{})
		handle(err)
	} else {
		handle(err)
	}

	startNFS(ctx, clientset, initNode)
}

func startNFS(ctx context.Context, clientset *kubernetes.Clientset, initNode string) {
	createNFSBackingPV(ctx, clientset, initNode)
	ssClient := clientset.AppsV1().StatefulSets(corev1.NamespaceDefault)
	var replicas int32 = 1
	var termination int64 = 10

	provisioner := "anrg.usc.edu/defer-nfs"

	ssSpec := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nfs-provisioner",
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "nfs-provisioner"},
			},
			ServiceName: "nfs-provisioner",
			Replicas:    &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "nfs-provisioner"},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            "nfs-provisioner",
					TerminationGracePeriodSeconds: &termination,
					Containers: []corev1.Container{
						{
							Name: "nfs-provisioner",
							//Image: "ghcr.io/kubernetes-sigs/nfs-ganesha:v3.5",
							Image: "gcr.io/k8s-staging-sig-storage/nfs-provisioner:arm64-linux-canary",
							Ports: []corev1.ContainerPort{
								{
									Name:          "nfs",
									ContainerPort: 2049,
								},
								{
									Name:          "nfs-udp",
									ContainerPort: 2049,
									Protocol:      corev1.Protocol(net.UDP),
								},
								{
									Name:          "nlockmgr",
									ContainerPort: 32803,
								},
								{
									Name:          "nlockmgr-udp",
									ContainerPort: 32803,
									Protocol:      corev1.Protocol(net.UDP),
								},
								{
									Name:          "mountd",
									ContainerPort: 2049,
								},
								{
									Name:          "mountd-udp",
									ContainerPort: 2049,
									Protocol:      corev1.Protocol(net.UDP),
								},
								{
									Name:          "rquotad",
									ContainerPort: 875,
								},
								{
									Name:          "rquotad-udp",
									ContainerPort: 875,
									Protocol:      corev1.Protocol(net.UDP),
								},
								{
									Name:          "rpcbind",
									ContainerPort: 111,
								},
								{
									Name:          "rpcbind-udp",
									ContainerPort: 111,
									Protocol:      corev1.Protocol(net.UDP),
								},
								{
									Name:          "statd",
									ContainerPort: 662,
								},
								{
									Name:          "statd-udp",
									ContainerPort: 662,
									Protocol:      corev1.Protocol(net.UDP),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"DAC_READ_SEARCH", "SYS_RESOURCE"},
								},
							},
							Args: []string{"-provisioner=" + provisioner},
							Env: []corev1.EnvVar{
								{
									Name: "POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "SERVICE_NAME",
									Value: "nfs-provisioner",
								},
								{
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
							ImagePullPolicy: corev1.PullAlways,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "nfs-volume",
									MountPath: "/export",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "nfs-volume",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "nfs-backing-pvc",
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := ssClient.Create(ctx, ssSpec, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		panic(err)
	}
}

// For a Local path PV, need to specify a node to place it on, so we just place it on the same node that the
// system init job ran on since we know that's a healthy node
func createNFSBackingPV(ctx context.Context, clientset *kubernetes.Clientset, initNode string) {
	fmt.Println("Creating NFS-backing PV")
	pvClient := clientset.CoreV1().PersistentVolumes()
	fs := corev1.PersistentVolumeFilesystem

	pvSpec := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nfs-backing-pv",
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceStorage: resource.MustParse("50Mi"),
			},
			VolumeMode: &fs,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			StorageClassName:              "nfs-backing-storage",
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				Local: &corev1.LocalVolumeSource{
					// The /var path is where the ext4 disk partition is located on the minikube nodes
					Path: "/var/nfs_backing_pv",
				},
			},
			NodeAffinity: &corev1.VolumeNodeAffinity{
				Required: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									// This will be the node name
									Key:      "kubernetes.io/hostname",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{initNode},
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := pvClient.Create(ctx, pvSpec, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		panic(err)
	}
}
