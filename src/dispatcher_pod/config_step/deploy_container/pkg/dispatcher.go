package deploy_pods

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func handle(e error) {
	if e != nil {
		panic(e)
	}
}

func DeployInferencePods(ctx context.Context, clientset *kubernetes.Clientset) {
	dir := "/nfs/model_config/partitions/"
	entries, _ := os.ReadDir(dir)
	nodes := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			nodes = append(nodes, e.Name())
		}
	}
	fmt.Println(nodes)

	var node, _ = os.ReadFile("/nfs/dispatcher_config/dispatcher_node.txt")
	dispatcherNode := string(node)

	deploymentClient := clientset.AppsV1().Deployments(corev1.NamespaceDefault)
	for _, n := range nodes {
		fp := filepath.Join(dir, n, "next_node.txt")
		nextNode, err := ioutil.ReadFile(fp)
		handle(err)

		// Each partition needs to have its own deployment (because the pod spec is different for each partition)
		var replicas int32 = 1
		var terminationGracePeriod int64 = 0
		partitionDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-partition", n),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas, // Each partitionDeployment only has 1 pod
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"task":          "inference",
						"type":          "compute",
						"assigned-node": n,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"task":          "inference",
							"type":          "compute",
							"assigned-node": n,
						},
					},
					Spec: corev1.PodSpec{
						TerminationGracePeriodSeconds: &terminationGracePeriod,
						Containers: []corev1.Container{
							{
								// Python inference container (same image for all pods)
								Name:  "inference-runtime",
								Image: "localhost:5000/inference_runtime:latest",
								// Build image to local container registry and pull
								ImagePullPolicy: corev1.PullAlways,
								Env: []corev1.EnvVar{
									{
										Name:  "NODE",
										Value: n,
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "nfs-volume",
										MountPath: "/nfs",
									},
									{
										Name:      "pipe-communication",
										MountPath: "/io",
									},
								},
							},
							{
								// Golang input/output socket
								Name:  "inference-sockets",
								Image: "localhost:5000/pod_inference_io:latest",
								// Build image to local container registry and pull
								ImagePullPolicy: corev1.PullAlways,
								Ports: []corev1.ContainerPort{{
									ContainerPort: 8080,
									Name:          "socket-io",
									Protocol:      "TCP",
								}},
								Env: []corev1.EnvVar{
									{
										Name:  "NODE",
										Value: n,
									},
									{
										Name:  "NEXT_NODE",
										Value: string(nextNode),
									},
									{
										Name:  "DISPATCHER_NAME",
										Value: dispatcherNode,
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "pipe-communication",
										MountPath: "/io",
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
							// NFS Server which holds all config data
							{
								Name: "nfs-volume",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "nfs",
									},
								},
							},
							// Communication between Golang sockets and Python inference runtime
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
						// Inference pod tries to be scheduled to where the placement algorithm told it to,
						// but if that node is unavailable then Kubernetes will still schedule the pod somewhere else
						Affinity: &corev1.Affinity{
							NodeAffinity: &corev1.NodeAffinity{
								PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
									{
										Weight: 1,
										Preference: corev1.NodeSelectorTerm{
											MatchExpressions: []corev1.NodeSelectorRequirement{
												{
													Key:      "kubernetes.io/hostname",
													Operator: corev1.NodeSelectorOpIn,
													Values:   []string{n},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Create deployment
		_, err = deploymentClient.Create(ctx, partitionDeployment, metav1.CreateOptions{})
		if err != nil && errors.IsAlreadyExists(err) {
			err := deploymentClient.Delete(ctx, fmt.Sprintf("%s-partition", n), metav1.DeleteOptions{})
			handle(err)
			_, err = deploymentClient.Create(ctx, partitionDeployment, metav1.CreateOptions{})
			handle(err)
		}
		fmt.Printf("Created deployment for %s\n", n)
	}
}

func DeployDispatcherInferencePod(ctx context.Context, clientset *kubernetes.Clientset) {
	deploymentClient := clientset.AppsV1().Deployments(corev1.NamespaceDefault)

	var node, _ = os.ReadFile("/nfs/dispatcher_config/dispatcher_node.txt")
	dispatcherNode := string(node)
	fmt.Printf("Dispatcher node: %s\n")

	var replicas int32 = 1
	var terminationGracePeriod int64 = 0

	dispatcherDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dispatcher-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"task":          "dispatcher_inference",
					"type":          "dispatcher",
					"assigned-node": dispatcherNode,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"task":          "dispatcher_inference",
						"type":          "dispatcher",
						"assigned-node": dispatcherNode,
					},
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriod,
					Containers: []corev1.Container{
						{
							// Python inference container (same image for all pods)
							Name:  "dispatch-inference-data",
							Image: "localhost:5000/dispatcher_inference_io:latest",
							// Build image to local container registry and pull
							ImagePullPolicy: corev1.PullAlways,
							// Finished inference from last compute node
							Ports: []corev1.ContainerPort{{
								ContainerPort: 8080,
								Name:          "socket-io",
								Protocol:      "TCP",
							}},
							Env: []corev1.EnvVar{
								{
									Name:  "NODE",
									Value: dispatcherNode,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								// Read dispatcher next node
								{
									Name:      "nfs-volume",
									MountPath: "/nfs",
								},
								// 1. Get data from process-inference-input container over FIFO
								// 2. Create the readiness check file
								{
									Name:      "pipe-communication",
									MountPath: "/io",
								},
							},
						},
						{
							// Process external inference input
							Name:  "process-inference-input",
							Image: "localhost:5000/process_inference_input:latest",
							// Build image to local container registry and pull
							ImagePullPolicy: corev1.PullAlways,
							VolumeMounts: []corev1.VolumeMount{
								// 1. Get data from dispatch-inference-data container over FIFO
								// 2. Look for the readiness check file
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
					// Dispatcher tries to be scheduled to where the placement algorithm told it to,
					// but if that node is unavailable then Kubernetes will still schedule the pod somewhere else
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
								{
									Weight: 1,
									Preference: corev1.NodeSelectorTerm{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/hostname",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{dispatcherNode},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	// Create deployment
	_, err := deploymentClient.Create(ctx, dispatcherDeployment, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		fmt.Println("Dispatcher deployment already created, deleting and re-creating")
		err = deploymentClient.Delete(ctx, "dispatcher-deployment", metav1.DeleteOptions{})
		handle(err)
		_, err = deploymentClient.Create(ctx, dispatcherDeployment, metav1.CreateOptions{})
		handle(err)
	}

	fmt.Println("Created dispatcher deployment")
}
