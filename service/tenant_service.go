package service

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/kenelite/gosphere-backend/model"
	"github.com/kenelite/gosphere-backend/pkg"
	"github.com/kenelite/gosphere-backend/repository"
)

type TenantService interface {
	CreateTenant(ctx context.Context, name string, description string) (*model.Tenant, error)
	GetTenant(ctx context.Context, id uint) (*model.Tenant, error)
	ListTenants(ctx context.Context) ([]model.Tenant, error)
	DeleteTenant(ctx context.Context, id uint) error
}

type tenantService struct {
	repo repository.TenantRepository
}

func NewTenantService(repo repository.TenantRepository) TenantService {
	return &tenantService{repo: repo}
}

func (s *tenantService) CreateTenant(ctx context.Context, name string, description string) (*model.Tenant, error) {
	ns := fmt.Sprintf("tenant-%s", name)
	if err := pkg.EnsureNamespace(ctx, ns); err != nil {
		return nil, err
	}
	// Minimal quota example; in real use read from request
	_ = pkg.ApplyResourceQuota(ctx, ns, "rq-default", map[corev1.ResourceName]string{
		corev1.ResourceCPU:    "2",
		corev1.ResourceMemory: "4Gi",
	})

	tenant := &model.Tenant{Name: name, Namespace: ns, Description: description}
	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *tenantService) GetTenant(ctx context.Context, id uint) (*model.Tenant, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *tenantService) ListTenants(ctx context.Context) ([]model.Tenant, error) {
	return s.repo.List(ctx)
}

func (s *tenantService) DeleteTenant(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}
