package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"notify-controller/activemq"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	MQ_HOST = "amqps://ulrrimlg:OzH3sTunX8kikcKKNnJCAr7vGiyPcPyq@lionfish.rmq.cloudamqp.com/ulrrimlg"
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

	factory := informers.NewSharedInformerFactory(clientset, 0)

	// Get the informer for the right resource, in this case a Pod
	informer := factory.Core().V1().Pods().Informer()

	stopper := make(chan struct{})
	defer close(stopper)
	defer runtime.HandleCrash()

	mq := activemq.NewActiveMQ(MQ_HOST)
	// This is the part where your custom code gets triggered based on the
	// event that the shared informer catches
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// When a new pod gets created
		AddFunc: func(obj interface{}) { fmt.Printf("Add not implemented") },
		// When a pod gets updated
		UpdateFunc: func(interface{}, interface{}) { fmt.Printf("update not implemented") },
		// When a pod gets deleted
		DeleteFunc: func(obj interface{}) {
			fmt.Printf("delete object")
			msg, err := json.Marshal(obj)
			if err != nil {

				panic(err.Error())
			}
			err = mq.Send("mychannel", msg)
			if err != nil {
				panic(err.Error())
			}
		},
	})
	go informer.Run(stopper)

	<-stopper

}
