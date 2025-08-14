package pkg

import (
	"context"
	"flag"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeClient kubernetes.Interface
var kubeCfg *rest.Config

func InitKubernetesClient() error {
	if kubeClient != nil {
		return nil
	}

	var cfg *rest.Config
	// Prefer in-cluster, fallback to kubeconfig
	var err error
	cfg, err = rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			if home, ok := os.LookupEnv("HOME"); ok {
				kubeconfig = fmt.Sprintf("%s/.kube/config", home)
			}
		}
		flag.Parse()
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return err
		}
	}

	c, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	kubeClient = c
	kubeCfg = cfg
	return nil
}

func Kube() kubernetes.Interface { return kubeClient }

func KubeRestConfig() *rest.Config { return kubeCfg }

// SetKubeClientForTest allows tests to inject a fake clientset
func SetKubeClientForTest(c kubernetes.Interface) { kubeClient = c }

func EnsureNamespace(ctx context.Context, name string) error {
	if name == "" {
		return nil
	}
	_, err := kubeClient.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	_, err = kubeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}, metav1.CreateOptions{})
	return err
}

func ApplyResourceQuota(ctx context.Context, namespace string, name string, hard map[corev1.ResourceName]string) error {
	if namespace == "" || name == "" {
		return nil
	}
	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       corev1.ResourceQuotaSpec{Hard: corev1.ResourceList{}},
	}
	for k, v := range hard {
		quantity := resourceQuantityFromString(v)
		quota.Spec.Hard[k] = quantity
	}
	_, err := kubeClient.CoreV1().ResourceQuotas(namespace).Create(ctx, quota, metav1.CreateOptions{})
	return err
}

// resourceQuantityFromString safely parses a quantity string. Falls back to 0 on error.
func resourceQuantityFromString(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		return resource.MustParse("0")
	}
	return q
}
