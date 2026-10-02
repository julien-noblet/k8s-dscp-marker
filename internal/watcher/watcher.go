// Package watcher watches the Pods scheduled on a single Kubernetes node and
// exposes the current set of DSCP marking rules derived from their
// annotations.
package watcher

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/julien-noblet/k8s-dscp-marker/internal/mangle"
)

// New returns a SharedIndexInformer restricted to the Pods scheduled on
// nodeName. resync is the period at which client-go replays Update events
// for every cached Pod; the caller uses those replayed events to re-apply
// DSCP rules periodically, so no separate polling ticker is needed.
//
// cache.NewListWatchFromClient already implements field-selector filtered
// list/watch against the API server, so there is no need to hand-roll one.
func New(client kubernetes.Interface, nodeName string, resync time.Duration) cache.SharedIndexInformer {
	selector := fields.OneTermEqualSelector("spec.nodeName", nodeName)
	listWatch := cache.NewListWatchFromClient(
		client.CoreV1().RESTClient(), "pods", metav1.NamespaceAll, selector,
	)

	return cache.NewSharedIndexInformer(
		listWatch,
		&corev1.Pod{},
		resync,
		cache.Indexers{},
	)
}

// RulesFromPods converts the Pods currently known to the informer into DSCP
// marking rules. Pods without the annotation or without an assigned IP are
// skipped.
func RulesFromPods(pods []*corev1.Pod, annotationKey string) []mangle.Rule {
	var rules []mangle.Rule
	for _, p := range pods {
		class, ok := p.Annotations[annotationKey]
		if !ok || class == "" {
			continue
		}
		if p.Status.PodIP == "" {
			continue
		}
		rules = append(rules, mangle.Rule{PodIP: p.Status.PodIP, Class: class})
	}
	return rules
}
