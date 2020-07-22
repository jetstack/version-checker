package main

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Load all auth plugins
	"k8s.io/client-go/tools/clientcmd"

	"github.com/joshvanl/version-checker/pkg/controller"
	"github.com/joshvanl/version-checker/pkg/metrics"
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

	metrics := metrics.New(logrus.NewEntry(logrus.New()))
	if err := metrics.Run(context.TODO(), ":0"); err != nil {
		logrus.Fatal(err)
	}

	c := controller.New(time.Second*3, metrics, kubeClient)
	if err := c.Run(context.TODO()); err != nil {
		logrus.Fatal(err)
	}
	//if err := controller.Run(context.TODO(), time.Minute*10, kubeClient); err != nil {
	//	logrus.Fatal(err)
	//}
}
