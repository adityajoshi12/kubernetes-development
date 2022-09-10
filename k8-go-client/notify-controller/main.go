package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"notify-controller/activemq"
	"path/filepath"
)

const (
	MQ_HOST = ""
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
		_ = fmt.Errorf("failed to config %s", err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	stopper := make(chan struct{})
	defer close(stopper)

	mq := activemq.NewActiveMQ(MQ_HOST)

	factory := informers.NewSharedInformerFactory(clientset, 0)

	informer := factory.Core().V1().Pods().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fmt.Println("add event")
		},
		UpdateFunc: func(obj1, obj2 interface{}) {
			fmt.Println("update event")
		},
		DeleteFunc: func(obj interface{}) {
			fmt.Println("delete event")
			msg, err := json.Marshal(obj)
			if err != nil {
				return
			}
			mq.Send(msg)
		},
	})

	go informer.Run(stopper)
	<-stopper
}
