// Command k8s-dscp-marker runs on every node of the cluster as a DaemonSet.
// It watches the Pods scheduled on its own node and, for every Pod carrying
// the configured annotation, marks its outgoing traffic with the requested
// DSCP class using a dedicated iptables mangle chain.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"k8s.io/client-go/rest"

	"github.com/julien-noblet/k8s-dscp-marker/internal/mangle"
	"github.com/julien-noblet/k8s-dscp-marker/internal/watcher"
)

const (
	defaultAnnotationKey = "dscp-marker.io/class"
	defaultChain         = "DSCP_MARKER"
	defaultResync        = 60 * time.Second
	// Debounce window: coalesces bursts of Pod events (e.g. at node boot)
	// into a single iptables reconciliation instead of one per event.
	debounce = 200 * time.Millisecond
)

func main() {
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig (defaults to in-cluster config; "+
			"set for local development, e.g. "+home+"/.kube/config)")
	}
	flag.Parse()

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatal("NODE_NAME environment variable is required (set it via the Downward API)")
	}

	annotationKey := envOr("ANNOTATION_KEY", defaultAnnotationKey)
	chain := envOr("CHAIN_NAME", defaultChain)

	client, err := buildClient(kubeconfig)
	if err != nil {
		log.Fatalf("building Kubernetes client: %v", err)
	}

	if err := mangle.EnsureChain(chain); err != nil {
		log.Fatalf("setting up iptables chain %s: %v", chain, err)
	}

	informer := watcher.New(client, nodeName, defaultResync)

	trigger := make(chan struct{}, 1)
	notify := func() {
		select {
		case trigger <- struct{}{}:
		default:
		}
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(interface{}) { notify() },
		UpdateFunc: func(interface{}, interface{}) { notify() },
		DeleteFunc: func(interface{}) { notify() },
	})

	stop := make(chan struct{})
	defer close(stop)
	go informer.Run(stop)

	if !cache.WaitForCacheSync(stop, informer.HasSynced) {
		log.Fatal("timed out waiting for the Pod cache to sync")
	}
	log.Printf("watching Pods on node %s, annotation %q, chain %s", nodeName, annotationKey, chain)

	for range trigger {
		time.Sleep(debounce)
		reconcile(informer, annotationKey, chain)
	}
}

func reconcile(informer cache.SharedIndexInformer, annotationKey, chain string) {
	var pods []*corev1.Pod
	for _, obj := range informer.GetIndexer().List() {
		if p, ok := obj.(*corev1.Pod); ok {
			pods = append(pods, p)
		}
	}

	rules := watcher.RulesFromPods(pods, annotationKey)
	if err := mangle.Reconcile(chain, rules); err != nil {
		log.Printf("reconciling chain %s: %v", chain, err)
		return
	}
	log.Printf("reconciled %s: %d pod(s) marked", chain, len(rules))
}

func buildClient(kubeconfig string) (kubernetes.Interface, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig: %w", err)
	}
	return kubernetes.NewForConfig(config)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
