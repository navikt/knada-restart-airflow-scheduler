package main

import (
	"context"
	"flag"
	"log"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const schedulerName = "airflow-scheduler"
const envLocal = "local"
const envProd = "inCluster"

func processNamespace(namespaceName string, deploymentName string, clientset *kubernetes.Clientset) {
	log.Printf("Processing namespace: %s", namespaceName)
	deploymentClient := clientset.AppsV1().Deployments(namespaceName)

	deployment, err := deploymentClient.Get(context.Background(), deploymentName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Deployment %s not found in namespace %s", deploymentName, namespaceName)
			return
		} else {
			log.Printf("Error getting deployment %s in namespace %s: %v", deploymentName, namespaceName, err)
			return
		}
	}

	// Trigger a restart by updating an annotation
	if deployment.Spec.Template.ObjectMeta.Annotations == nil {
		deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.ObjectMeta.Annotations["knada/restartedAt"] = time.Now().Format(time.RFC3339)

	// Update the deployment
	_, err = deploymentClient.Update(context.Background(), deployment, metav1.UpdateOptions{})
	if err != nil {
		log.Printf("Error updating deployment %s in namespace %s: %v", deploymentName, namespaceName, err)
		return
	}

	log.Printf("Deployment %s in namespace %s restarted successfully\n", deploymentName, namespaceName)
}

func main() {
	var deploymentName string
	var env string

	flag.StringVar(&deploymentName, "deployment", schedulerName, "Name of the airflow scheduler deployment to restart")
	flag.StringVar(&env, "env", envLocal, "Environment running in")
	flag.Parse()

	var config *rest.Config
	var err error

	if env == envLocal {
		if home := homedir.HomeDir(); home != "" {
			configPath := filepath.Join(home, ".kube", "config")
			config, err = clientcmd.BuildConfigFromFlags("", configPath)
			if err != nil {
				log.Fatalf("Error building kubeconfig: %s", err.Error())
				return
			}
		} else {
			log.Fatalf("Home directory not found")
			return
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("Error building in-cluster config: %s", err.Error())
			return
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating kubernetes client: %s", err.Error())
		return
	}

	// Process all namespaces
	namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error getting namespaces: %s", err.Error())
		return
	}

	for _, ns := range namespaces.Items {
		namespaceName := ns.Name
		processNamespace(namespaceName, deploymentName, clientset)
	}
}
