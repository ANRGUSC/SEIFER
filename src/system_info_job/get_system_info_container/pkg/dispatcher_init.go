package system_info

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/net"
)

func InitDispatcher(ctx context.Context, clientset *kubernetes.Clientset, dispatcherName string) {
	fmt.Println("Initializing dispatcher")
	deploymentClient := clientset.AppsV1().Deployments(metav1.NamespaceDefault)

	// Use existing configmap
	/*cmd := exec.Command("/bin/sh", "/root/dispatcher_configmap.sh")
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	handle(err)*/

	var replicas int32 = 1
	dispatcherDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dispatcher-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":           "dispatcher",
					"assigned-node": "dispatcher",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":           "dispatcher",
						"assigned-node": "dispatcher",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "dispatcher-partitioner",
							Image: "ghcr.io/dat-boi-arjun/dispatcher_partitioner:latest",
							Env: []corev1.EnvVar{
								{
									Name:  "DISPATCHER_NAME",
									Value: dispatcherName,
								},
							},
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
						{
							Name:  "dispatcher-deploy-pods",
							Image: "ghcr.io/dat-boi-arjun/deploy_pods:latest",
							VolumeMounts: []corev1.VolumeMount{
								// To read the partitions directory and find the node names
								{
									Name:      "nfs-volume",
									MountPath: "/nfs",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							// Python inference container (same image for all pods)
							Name:  "dispatch-inference-data",
							Image: "ghcr.io/dat-boi-arjun/dispatcher_inference_io:latest",
							VolumeMounts: []corev1.VolumeMount{
								// Read dispatcher next node
								{
									Name:      "nfs-volume",
									MountPath: "/nfs",
								},
								// Get data from process-inference-input container over FIFO
								{
									Name:      "pipe-communication",
									MountPath: "/io",
								},
							},
							// Finished inference from last compute node
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
						},
						{
							// Process external inference input
							Name:  "process-inference-input",
							Image: "ghcr.io/dat-boi-arjun/process_inference_input:latest",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "pipe-communication",
									MountPath: "/io",
								},
							},
							// Input from external source
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 3000,
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
					RestartPolicy:      corev1.RestartPolicyAlways,
					DNSPolicy:          "ClusterFirst",
					ServiceAccountName: "defer-admin-account",
					// Dispatcher gets scheduled to wherever the system-init-job was scheduled
					NodeName: dispatcherName,
				},
			},
		},
	}

	// Create deployment
	_, err := deploymentClient.Create(ctx, dispatcherDeployment, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		err = deploymentClient.Delete(ctx, "dispatcher-deployment", metav1.DeleteOptions{})
		handle(err)
		_, err = deploymentClient.Create(ctx, dispatcherDeployment, metav1.CreateOptions{})
		handle(err)
	}

	// Use existing cluster NFS
	//startNFS(ctx, clientset, dispatcherName)
}

func startNFS(ctx context.Context, clientset *kubernetes.Clientset, dispatcherName string) {
	createDispatcherPV(ctx, clientset, dispatcherName)
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
									ClaimName: "dispatcher-nfs-pvc",
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

func createDispatcherPV(ctx context.Context, clientset *kubernetes.Clientset, dispatcherName string) {
	fmt.Println("Creating dispatcher PV")
	pvClient := clientset.CoreV1().PersistentVolumes()
	fs := corev1.PersistentVolumeFilesystem

	pvSpec := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dispatcher-nfs-pv",
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
			StorageClassName:              "dispatcher-nfs-storage",
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				Local: &corev1.LocalVolumeSource{
					// The /var path is where the ext4 disk partition is located on the minikube nodes
					Path: "/var/dispatcher_pv",
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
									Operator: "In",
									Values:   []string{dispatcherName},
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
