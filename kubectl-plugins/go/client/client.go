package client

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

func K8Client() *kubernetes.Clientset {

	config, err := genericclioptions.NewConfigFlags(true).ToRESTConfig()
	if err != nil {
		panic(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return client

}
