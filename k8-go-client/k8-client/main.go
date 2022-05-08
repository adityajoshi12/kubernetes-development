package main

import (
	"context"
	"flag"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

func main() {

	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Errorf(err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	list, err := clientset.CoreV1().Pods("").List(context.Background(), v1.ListOptions{})
	if err != nil {
		return
	}

	fmt.Printf("total pods %d\n", len(list.Items))
	for _, pod := range list.Items {
		fmt.Printf("%s : %s\n", pod.Name, pod.Namespace)
	}
	pod, err := clientset.CoreV1().Pods("kube-system").Get(context.Background(), "kube-apiserver-kind-control-plane", v1.GetOptions{})
	if err != nil {
		panic(err.Error())
		return
	}
	fmt.Println(pod.Labels)
}
