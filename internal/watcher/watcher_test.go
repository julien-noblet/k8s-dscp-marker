package watcher

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/julien-noblet/k8s-dscp-marker/internal/mangle"
)

func pod(name, ip string, annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Annotations: annotations},
		Status:     corev1.PodStatus{PodIP: ip},
	}
}

func TestRulesFromPods(t *testing.T) {
	const key = "dscp-marker.io/class"

	pods := []*corev1.Pod{
		pod("voice", "10.0.0.1", map[string]string{key: "EF"}),
		pod("no-annotation", "10.0.0.2", nil),
		pod("empty-annotation", "10.0.0.3", map[string]string{key: ""}),
		pod("not-scheduled-yet", "", map[string]string{key: "CS1"}),
	}

	got := RulesFromPods(pods, key)
	want := []mangle.Rule{{PodIP: "10.0.0.1", Class: "EF"}}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("RulesFromPods() = %v, want %v", got, want)
	}
}
