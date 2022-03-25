package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GetClientSet creates a k8s client from the cluster config.
// Must be run inside a running pod
func GetClientSet() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}
