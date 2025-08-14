package router

import (
	"github.com/gin-gonic/gin"

	"github.com/kenelite/gosphere-backend/controller"
)

func Register(r *gin.Engine) {
	api := r.Group("/api/v1")

	// Tenant routes
	api.POST("/tenants", controller.CreateTenant)
	api.GET("/tenants", controller.ListTenants)
	api.GET("/tenants/:id", controller.GetTenant)
	api.DELETE("/tenants/:id", controller.DeleteTenant)

	// Webhooks
	r.POST("/webhooks/github", controller.GitHubWebhook)

	// Deployment and rollout controls
	api.POST("/deploy/canary/weight", controller.SetCanaryWeight)
	api.POST("/deploy/swimlane/header", controller.SetSwimlaneHeader)
	api.POST("/deploy/bluegreen/switch", controller.SwitchBlueGreen)

	// Autoscaling
	api.POST("/autoscaling/hpa", controller.EnsureHPA)
	api.POST("/autoscaling/vpa", controller.EnsureVPA)

	// Harbor gate for Argo sync
	api.POST("/cicd/harbor-gate-sync", controller.HarborGateSync)
}
