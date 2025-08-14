package repository

import (
	"context"

	"github.com/kenelite/gosphere-backend/model"
	"github.com/kenelite/gosphere-backend/pkg"
)

type TenantRepository interface {
	Create(ctx context.Context, t *model.Tenant) error
	FindByID(ctx context.Context, id uint) (*model.Tenant, error)
	List(ctx context.Context) ([]model.Tenant, error)
	Delete(ctx context.Context, id uint) error
}

type tenantRepositoryGorm struct{}

func NewTenantRepository() TenantRepository { return &tenantRepositoryGorm{} }

func (r *tenantRepositoryGorm) Create(ctx context.Context, t *model.Tenant) error {
	return pkg.DB().WithContext(ctx).Create(t).Error
}

func (r *tenantRepositoryGorm) FindByID(ctx context.Context, id uint) (*model.Tenant, error) {
	var t model.Tenant
	if err := pkg.DB().WithContext(ctx).First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *tenantRepositoryGorm) List(ctx context.Context) ([]model.Tenant, error) {
	var items []model.Tenant
	if err := pkg.DB().WithContext(ctx).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *tenantRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return pkg.DB().WithContext(ctx).Delete(&model.Tenant{}, id).Error
}
