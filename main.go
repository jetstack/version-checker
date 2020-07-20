package main

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Load all auth plugins
	"k8s.io/client-go/tools/clientcmd"

	"github.com/joshvanl/version-checker/pkg/controller"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		logrus.Fatal(err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Fatal(err)
	}

	if err := controller.Run(context.TODO(), kubeClient); err != nil {
		logrus.Fatal(err)
	}
}
