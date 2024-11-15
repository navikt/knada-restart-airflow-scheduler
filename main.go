package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const schedulerName = "airflow-scheduler"
const envLocal = "local"
const envProd = "inCluster"

func getNamespaceFromConfig(kubeconfigPath string) (string, error) {
	configLoadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	configOverrides := &clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(configLoadingRules, configOverrides)
	namespace, _, err := clientConfig.Namespace()
	return namespace, err
}

func main() {
	var deploymentName string
	var namespace string
	var env string

	flag.StringVar(&deploymentName, "deployment", schedulerName, "Name of the airflow scheduler deployment to restart")
	flag.StringVar(&namespace, "namespace", "", "Namespace of the airflow scheduler deployment")
	flag.StringVar(&env, "env", envLocal, "Environment running in")
	flag.Parse()

	var config *rest.Config
	var err error

	if env == envLocal {
		if home := homedir.HomeDir(); home != "" {
			configPath := filepath.Join(home, ".kube", "config")
			config, err = clientcmd.BuildConfigFromFlags("", configPath)
			if err != nil {
				panic(err.Error())
			}
			if namespace == "" {
				namespace, err = getNamespaceFromConfig(configPath)
				if err != nil {
					panic(err.Error())
				}
			}
		} else {
			panic("Home directory not found")
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	log.Printf("Restarting airflow scheduler in namespace %s, deploymentName %s\n", namespace, deploymentName)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	deploymentClient := clientset.AppsV1().Deployments(namespace)

	// Get the deployment
	deployment, err := deploymentClient.Get(context.Background(), deploymentName, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	// Trigger a restart by updating an annotation
	if deployment.Spec.Template.ObjectMeta.Annotations == nil {
		deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.ObjectMeta.Annotations["knada/restartedAt"] = time.Now().Format(time.RFC3339)

	// Update the deployment
	_, err = deploymentClient.Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Deployment %s in namespace %s restarted successfully\n", deploymentName, namespace)
}
