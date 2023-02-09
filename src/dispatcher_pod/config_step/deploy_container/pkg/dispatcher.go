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

	deploymentClient := clientset.AppsV1().Deployments(corev1.NamespaceDefault)
	for _, n := range nodes {
		fp := filepath.Join(dir, n, "next_node.txt")
		nextNode, err := ioutil.ReadFile(fp)
		handle(err)

		// Each partition needs to have its own deployment (because the pod spec is different for each partition)
		var replicas int32 = 1
		partitionDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("node-%s-partition", n),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas, // Each partitionDeployment only has 1 pod
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"task":          "inference",
						"assigned-node": n,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"task":          "inference",
							"assigned-node": n,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								// Python inference container (same image for all pods)
								Name:            "inference-runtime",
								Image:           "ghcr.io/dat-boi-arjun/inference_runtime:latest",
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
								Image: "ghcr.io/dat-boi-arjun/pod_inference_io",
								Ports: []corev1.ContainerPort{{
									ContainerPort: 8080,
									Name:          "socket-io",
									Protocol:      "TCP",
								}},
								Env: []corev1.EnvVar{
									{
										Name:  "NEXT_NODE",
										Value: string(nextNode),
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
						ServiceAccountName: "default",
						NodeName:           n, // The node we want to pod to be scheduled on
					},
				},
			},
		}

		// Create deployment
		_, err = deploymentClient.Create(ctx, partitionDeployment, metav1.CreateOptions{})
		if err != nil && errors.IsAlreadyExists(err) {
			err := deploymentClient.Delete(ctx, fmt.Sprintf("node-%s-partition-deployment", n), metav1.DeleteOptions{})
			handle(err)
			_, err = deploymentClient.Create(ctx, partitionDeployment, metav1.CreateOptions{})
			handle(err)
		}
		fmt.Printf("Created deployment for %s\n", n)
	}
}

func DeployDispatcherInferencePod(ctx context.Context, clientset *kubernetes.Clientset) {
	deploymentClient := clientset.AppsV1().Deployments(corev1.NamespaceDefault)

	var node, _ = ioutil.ReadFile("/nfs/dispatcher_config/dispatcher_node.txt")
	dispatcherNode := string(node)

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
					NodeName: dispatcherNode,
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
}
