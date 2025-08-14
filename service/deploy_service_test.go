package service

import (
	"context"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kenelite/gosphere-backend/pkg"
)

func TestSwitchBlueGreen(t *testing.T) {
	c := fake.NewSimpleClientset()
	pkg.SetKubeClientForTest(c)
	ns := "default"
	ing := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "ing", Namespace: ns}}
	_, _ = c.NetworkingV1().Ingresses(ns).Create(context.Background(), ing, metav1.CreateOptions{})

	if err := SwitchBlueGreen(context.Background(), ns, "ing", "", "/", "svc-green"); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestConfigureCanaryWeight(t *testing.T) {
	c := fake.NewSimpleClientset()
	pkg.SetKubeClientForTest(c)
	ns := "default"

	// ensure canary ingress is created
	if err := ConfigureCanaryWeight(context.Background(), ns, "ing", "ing-canary", "example.com", "/", "svc", "svc-canary", 10); err != nil {
		t.Fatalf("err: %v", err)
	}
	actions := c.Actions()
	if len(actions) == 0 {
		t.Fatal("expected client actions")
	}
	// at least one create or update should be recorded
	found := false
	for _, a := range actions {
		if a.GetVerb() == "create" || a.GetVerb() == "update" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected create/update actions")
	}
}
