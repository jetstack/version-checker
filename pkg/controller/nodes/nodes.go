package nodes

import (
	"github.com/jetstack/version-checker/pkg/checker/architecture"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// NodeInformer is the wrapper for the k8s Node Informer
type NodeInformer struct {
	log   *logrus.Entry
	nodes *architecture.NodeMap
}

// New returns a new instance of NodeInformer
func New(log *logrus.Entry, nodes *architecture.NodeMap) *NodeInformer {
	return &NodeInformer{
		log:   log.WithField("module", "controller_node"),
		nodes: nodes,
	}
}

// Register returns the node informer with event handler set to update node architecture map
func (c *NodeInformer) Register(sharedInformerFactory informers.SharedInformerFactory) func() bool {
	nodeInformer := sharedInformerFactory.Core().V1().Nodes().Informer()
	nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Add node info and data to the map
			err := c.nodes.Add(obj.(*corev1.Node))
			if err != nil {
				c.log.Errorf("error adding the node %q to architecture map: %s", obj.(*corev1.Node), err)
				return
			}
		},
		UpdateFunc: func(old, new interface{}) {
			// override the map
			err := c.nodes.Add(new.(*corev1.Node))
			if err != nil {
				c.log.Errorf("error updating the node %q in architecture map: %s", old.(*corev1.Node), err)
				return
			}
		},
		DeleteFunc: func(obj interface{}) {
			// remove node from the map
			c.nodes.Delete((obj.(*corev1.Node)).Name)
		},
	})

	return nodeInformer.HasSynced
}
