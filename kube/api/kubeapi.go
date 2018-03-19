package kubeapi

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var clientSet *kubernetes.Clientset
var namespace string

func Setup(namespaceParam string) *kubernetes.Clientset {

	var kubeconfig string
	var homeDir string

	if homeDir = os.Getenv("HOME"); homeDir == "" {
		fmt.Println("=> Oh no! Couldn't figure out what your homedir is, please set the environment variable $HOME.")
		os.Exit(1)
	}

	kubeconfig = filepath.Join(homeDir, ".kube", "config")

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	clientSet = clientset
	namespace = namespaceParam
	return clientset
}

func GetSingleDeployment(name string) *v1beta1.Deployment {
	deployment, _ := clientSet.
		ExtensionsV1beta1().Deployments(namespace).
		Get(name, metav1.GetOptions{})
	// Return even if nil
	return deployment
}

func UpdateDeployment(name string, callback func(*v1beta1.Deployment)) *v1beta1.Deployment {

	var deployment *v1beta1.Deployment

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Retrieve the latest version of Deployment before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
		// result, getErr := deploymentsClient.Get("demo-deployment", metav1.GetOptions{})
		deployment = GetSingleDeployment(name)
		callback(deployment)
		_, updateErr := clientSet.ExtensionsV1beta1().Deployments(namespace).
			Update(deployment)
		return updateErr
	})
	if retryErr != nil {
		panic(fmt.Errorf("Update failed: %v", retryErr))
	}
	fmt.Printf("=> Updated deployment %s.\n", deployment.Name)

	return deployment
}

func AddDeploymentLabel(deployment *v1beta1.Deployment, key string, value string) {
	existingLabels := deployment.GetLabels()
	existingLabels[key] = value
}

func RemoveDeploymentLabel(deployment *v1beta1.Deployment, key string) {
	existingLabels := deployment.GetLabels()
	delete(existingLabels, key)
}

func DeleteDeployment(deployment *v1beta1.Deployment) {
	deletePolicy := metav1.DeletePropagationForeground

	if err := clientSet.
		ExtensionsV1beta1().Deployments(namespace).
		Delete(deployment.Name, &metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}); err != nil {
		panic(err.Error())
	}
}

func DeleteService(service *v1.Service) {
	if err := clientSet.CoreV1().Services(namespace).
		Delete(service.Name, nil); err != nil {
		panic(err.Error())
	}
}

func DeleteSecret(secret *v1.Secret) {
	if err := clientSet.CoreV1().Secrets(namespace).
		Delete(secret.Name, nil); err != nil {
		panic(err.Error())
	}
}

func DeleteIngress(ingress *v1beta1.Ingress) {
	if err := clientSet.ExtensionsV1beta1().Ingresses(namespace).
		Delete(ingress.Name, nil); err != nil {
		panic(err.Error())
	}
}

func ListDeployments(labelFilter map[string]string) *v1beta1.DeploymentList {

	label := labels.Set(labelFilter)

	deployments, err := clientSet.
		ExtensionsV1beta1().Deployments(namespace).
		List(metav1.ListOptions{LabelSelector: label.String()})

	if err != nil {
		panic(err.Error())
	}

	return deployments
	// // Examples for error handling:
	// // - Use helper functions like e.g. errors.IsNotFound()
	// // - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	// _, err = clientset.CoreV1().Pods("default").Get("example-xxxxx", metav1.GetOptions{})
	// if errors.IsNotFound(err) {
	// 	fmt.Printf("Pod not found\n")
	// } else if statusError, isStatus := err.(*errors.StatusError); isStatus {
	// 	fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
	// } else if err != nil {
	// 	panic(err.Error())
	// } else {
	// 	fmt.Printf("Found pod\n")
	// }

	// time.Sleep(10 * time.Second)

}
