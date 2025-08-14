package pkg

import (
	"context"
	"fmt"

	appv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdclient "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var argoCS argocdclient.Interface

// InitArgoClient initializes the Argo CD typed client using the Kubernetes rest.Config.
func InitArgoClient() error {
	if argoCS != nil {
		return nil
	}
	cfg := KubeRestConfig()
	if cfg == nil {
		return fmt.Errorf("kube rest config is nil; call InitKubernetesClient first")
	}
	cs, err := argocdclient.NewForConfig(cfg)
	if err != nil {
		return err
	}
	argoCS = cs
	return nil
}

func Argo() argocdclient.Interface { return argoCS }

type ArgoApplicationSpec struct {
	Name                 string
	Namespace            string // namespace where Argo CD manages apps (argocd namespace)
	Project              string
	RepoURL              string
	TargetRevision       string
	Path                 string
	DestinationServer    string // e.g. https://kubernetes.default.svc
	DestinationNamespace string
	AutomatedSync        bool
}

// EnsureApplication creates or updates an Argo CD Application CR.
func EnsureApplication(ctx context.Context, spec ArgoApplicationSpec) (*appv1.Application, error) {
	if err := InitArgoClient(); err != nil {
		return nil, err
	}
	apps := argoCS.ArgoprojV1alpha1().Applications(spec.Namespace)

	desired := &appv1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
		},
		Spec: appv1.ApplicationSpec{
			Project: spec.Project,
			Source: &appv1.ApplicationSource{
				RepoURL:        spec.RepoURL,
				TargetRevision: spec.TargetRevision,
				Path:           spec.Path,
			},
			Destination: appv1.ApplicationDestination{
				Server:    spec.DestinationServer,
				Namespace: spec.DestinationNamespace,
			},
		},
	}
	if spec.AutomatedSync {
		desired.Spec.SyncPolicy = &appv1.SyncPolicy{Automated: &appv1.SyncPolicyAutomated{Prune: true, SelfHeal: true}}
	}

	existing, err := apps.Get(ctx, spec.Name, metav1.GetOptions{})
	if err == nil {
		// Update existing
		existing.Spec = desired.Spec
		return apps.Update(ctx, existing, metav1.UpdateOptions{})
	}
	// Create new
	return apps.Create(ctx, desired, metav1.CreateOptions{})
}

// TriggerSync sets an Operation on the Application to start a sync.
func TriggerSync(ctx context.Context, name, namespace string) error {
	if err := InitArgoClient(); err != nil {
		return err
	}
	apps := argoCS.ArgoprojV1alpha1().Applications(namespace)
	app, err := apps.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	op := &appv1.Operation{Sync: &appv1.SyncOperation{}}
	app.Operation = op
	_, err = apps.Update(ctx, app, metav1.UpdateOptions{})
	return err
}

type ApplicationStatus struct {
	Sync   string
	Health string
}

func GetApplicationStatus(ctx context.Context, name, namespace string) (*ApplicationStatus, error) {
	if err := InitArgoClient(); err != nil {
		return nil, err
	}
	app, err := argoCS.ArgoprojV1alpha1().Applications(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return &ApplicationStatus{
		Sync:   string(app.Status.Sync.Status),
		Health: string(app.Status.Health.Status),
	}, nil
}
