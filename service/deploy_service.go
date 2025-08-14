package service

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kenelite/gosphere-backend/pkg"
)

// EnsureHPA creates or updates an HPA for a target Deployment
func EnsureHPA(ctx context.Context, namespace string, name string, targetRef appsv1.Deployment, min, max int32, cpuUtilizationPercent int32) error {
	c := pkg.Kube().AutoscalingV2().HorizontalPodAutoscalers(namespace)
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       targetRef.Name,
				APIVersion: "apps/v1",
			},
			MinReplicas: &min,
			MaxReplicas: max,
			Metrics: []autoscalingv2.MetricSpec{{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name:   "cpu",
					Target: autoscalingv2.MetricTarget{Type: autoscalingv2.UtilizationMetricType, AverageUtilization: &cpuUtilizationPercent},
				},
			}},
		},
	}
	if _, err := c.Get(ctx, name, metav1.GetOptions{}); err == nil {
		_, err = c.Update(ctx, hpa, metav1.UpdateOptions{})
		return err
	}
	_, err := c.Create(ctx, hpa, metav1.CreateOptions{})
	return err
}

// EnsureVPA creates or updates a VerticalPodAutoscaler using dynamic client (CRD autoscaling.k8s.io/v1)
func EnsureVPA(ctx context.Context, namespace string, name string, targetDeployment string) error {
	cfg := pkg.KubeRestConfig()
	if cfg == nil {
		return fmt.Errorf("no kube rest config")
	}
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}
	gvr := schema.GroupVersionResource{Group: "autoscaling.k8s.io", Version: "v1", Resource: "verticalpodautoscalers"}
	vpa := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "autoscaling.k8s.io/v1",
			"kind":       "VerticalPodAutoscaler",
			"metadata":   map[string]any{"name": name, "namespace": namespace},
			"spec": map[string]any{
				"targetRef":    map[string]any{"apiVersion": "apps/v1", "kind": "Deployment", "name": targetDeployment},
				"updatePolicy": map[string]any{"updateMode": "Auto"},
			},
		},
	}
	res := dc.Resource(gvr).Namespace(namespace)
	if _, err := res.Get(ctx, name, metav1.GetOptions{}); err == nil {
		_, err = res.Update(ctx, vpa, metav1.UpdateOptions{})
		return err
	}
	_, err = res.Create(ctx, vpa, metav1.CreateOptions{})
	return err
}

// ConfigureCanaryWeight ensures a canary Ingress and sets weight using NGINX ingress annotations
func ConfigureCanaryWeight(ctx context.Context, namespace string, baseIngress string, canaryIngress string, host string, path string, service string, canaryService string, weight int) error {
	if weight < 0 {
		weight = 0
	}
	if weight > 100 {
		weight = 100
	}
	ingClient := pkg.Kube().NetworkingV1().Ingresses(namespace)
	// base ingress should route to stable service; canary ingress routes to canary service with canary annotations
	canary := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      canaryIngress,
			Namespace: namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/canary":        "true",
				"nginx.ingress.kubernetes.io/canary-weight": fmt.Sprintf("%d", weight),
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: host,
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
					Path:     path,
					PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
					Backend:  networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: canaryService, Port: networkingv1.ServiceBackendPort{Number: 80}}},
				}}}},
			}},
		},
	}
	if _, err := ingClient.Get(ctx, canaryIngress, metav1.GetOptions{}); err == nil {
		_, err = ingClient.Update(ctx, canary, metav1.UpdateOptions{})
		return err
	}
	_, err := ingClient.Create(ctx, canary, metav1.CreateOptions{})
	return err
}

// ConfigureSwimlaneByHeader sets canary-by-header to route requests with a header to canary service
func ConfigureSwimlaneByHeader(ctx context.Context, namespace string, canaryIngress string, header string, headerValue string) error {
	ingClient := pkg.Kube().NetworkingV1().Ingresses(namespace)
	ing, err := ingClient.Get(ctx, canaryIngress, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if ing.Annotations == nil {
		ing.Annotations = map[string]string{}
	}
	ing.Annotations["nginx.ingress.kubernetes.io/canary-by-header"] = header
	ing.Annotations["nginx.ingress.kubernetes.io/canary-by-header-value"] = headerValue
	_, err = ingClient.Update(ctx, ing, metav1.UpdateOptions{})
	return err
}

// SwitchBlueGreen updates the base ingress backend to point all traffic to the target service (blue or green)
func SwitchBlueGreen(ctx context.Context, namespace string, ingressName string, host string, path string, targetService string) error {
	ingClient := pkg.Kube().NetworkingV1().Ingresses(namespace)
	ing, err := ingClient.Get(ctx, ingressName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// find matching rule/path, otherwise fallback to first path
	updated := false
	for ri := range ing.Spec.Rules {
		rule := &ing.Spec.Rules[ri]
		if host != "" && rule.Host != host {
			continue
		}
		if rule.IngressRuleValue.HTTP == nil {
			continue
		}
		for pi := range rule.IngressRuleValue.HTTP.Paths {
			p := &rule.IngressRuleValue.HTTP.Paths[pi]
			if path == "" || p.Path == path {
				if p.Backend.Service == nil {
					p.Backend.Service = &networkingv1.IngressServiceBackend{}
				}
				p.Backend.Service.Name = targetService
				// keep port number as-is if set; else default to 80
				if p.Backend.Service.Port.Number == 0 && p.Backend.Service.Port.Name == "" {
					p.Backend.Service.Port = networkingv1.ServiceBackendPort{Number: 80}
				}
				updated = true
				break
			}
		}
		if updated {
			break
		}
	}
	// fallback if no rules
	if !updated {
		// initialize minimal spec
		pt := networkingv1.PathTypePrefix
		ing.Spec.Rules = []networkingv1.IngressRule{{
			Host: host,
			IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
				Path:     path,
				PathType: &pt,
				Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
					Name: targetService,
					Port: networkingv1.ServiceBackendPort{Number: 80},
				}},
			}}}},
		}}
	}
	_, err = ingClient.Update(ctx, ing, metav1.UpdateOptions{})
	return err
}
