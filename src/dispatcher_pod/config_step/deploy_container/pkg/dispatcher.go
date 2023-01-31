package deploy_pods

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func handle(e error) {
	if e != nil {
		panic(e)
	}
}

func DeployInferencePods(ctx context.Context) {
	config, err := rest.InClusterConfig()
	handle(err)

	clientset, err := kubernetes.NewForConfig(config)
	handle(err)

	// While testing, don't use NFS
	/*dir := "/nfs/model_config/partitions/"
	entries, _ := os.ReadDir(dir)
	nodes := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			nodes = append(nodes, e.Name())
		}
	}
	fmt.Println(nodes)*/
	// Dispatcher is on the "minikube" node, other nodes are compute nodes
	nodes := []string{"minikube-m02", "minikube-m03", "minikube-m04", "minikube-m05"}

	deploymentClient := clientset.AppsV1().Deployments(corev1.NamespaceDefault)
	for i, n := range nodes {
		/*fp := filepath.Join(dir, n, "next_node.txt")
		nextNode, err := ioutil.ReadFile(fp)
		handle(err)*/

		var nextNode string
		if n == "minikube-m05" {
			nextNode = "dispatcher"
		} else {
			nextNode = nodes[i+1]
		}

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
								Name:  "inference-runtime",
								Image: "ghcr.io/dat-boi-arjun/inference_runtime:latest",
								Env: []corev1.EnvVar{
									{
										Name:  "NODE",
										Value: n,
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									// For testing no NFS
									/*{
										Name:      "nfs-volume",
										MountPath: "/nfs",
									},*/
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
							// For testing don't use NFS
							// NFS Server which holds all config data
							/*{
								Name: "nfs-volume",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "nfs",
									},
								},
							},*/
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
