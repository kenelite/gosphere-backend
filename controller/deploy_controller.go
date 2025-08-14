package controller

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kenelite/gosphere-backend/pkg"
	"github.com/kenelite/gosphere-backend/service"
)

type GitHubWebhookResp struct {
	Status string `json:"status"`
}

func GitHubWebhook(c *gin.Context) {
	event, payload, body, err := pkg.HandleGitHubWebhook(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	// Only handle push for now: ensure/update Argo Application + trigger sync
	if event == "push" && payload != nil {
		// Simplified mapping: branch from ref, repo from full name
		branch := payload.Ref
		// ref format: refs/heads/<branch>
		if len(branch) > 11 {
			branch = branch[11:]
		}
		// Defaults; in practice map from DB/config
		spec := pkg.ArgoApplicationSpec{
			Name:                 "app-" + branch,
			Namespace:            "argocd",
			Project:              "default",
			RepoURL:              payload.Repository.CloneURL,
			TargetRevision:       branch,
			Path:                 ".",
			DestinationServer:    "https://kubernetes.default.svc",
			DestinationNamespace: "default",
			AutomatedSync:        true,
		}
		if _, err := pkg.EnsureApplication(c.Request.Context(), spec); err == nil {
			_ = pkg.TriggerSync(c.Request.Context(), spec.Name, spec.Namespace)
		}
	}
	_ = body // available for future event types
	c.JSON(http.StatusOK, GitHubWebhookResp{Status: "ok"})
}

type CanaryWeightRequest struct {
	Namespace     string `json:"namespace" binding:"required"`
	BaseIngress   string `json:"baseIngress" binding:"required"`
	CanaryIngress string `json:"canaryIngress" binding:"required"`
	Host          string `json:"host" binding:"required"`
	Path          string `json:"path" binding:"required"`
	StableService string `json:"stableService" binding:"required"`
	CanaryService string `json:"canaryService" binding:"required"`
	Weight        int    `json:"weight" binding:"min=0,max=100"`
}

func SetCanaryWeight(c *gin.Context) {
	var req CanaryWeightRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := service.ConfigureCanaryWeight(c.Request.Context(), req.Namespace, req.BaseIngress, req.CanaryIngress, req.Host, req.Path, req.StableService, req.CanaryService, req.Weight); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type SwimlaneHeaderRequest struct {
	Namespace     string `json:"namespace" binding:"required"`
	CanaryIngress string `json:"canaryIngress" binding:"required"`
	Header        string `json:"header" binding:"required"`
	Value         string `json:"value" binding:"required"`
}

func SetSwimlaneHeader(c *gin.Context) {
	var req SwimlaneHeaderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := service.ConfigureSwimlaneByHeader(c.Request.Context(), req.Namespace, req.CanaryIngress, req.Header, req.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type HPARequest struct {
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Deploy    string `json:"deployment" binding:"required"`
	Min       int32  `json:"min"`
	Max       int32  `json:"max" binding:"required"`
	CPU       int32  `json:"cpuUtilizationPercent"`
}

func EnsureHPA(c *gin.Context) {
	var req HPARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dep, err := pkg.Kube().AppsV1().Deployments(req.Namespace).Get(context.Background(), req.Deploy, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Min == 0 {
		req.Min = 1
	}
	if req.CPU == 0 {
		req.CPU = 60
	}
	if err := service.EnsureHPA(c.Request.Context(), req.Namespace, req.Name, *dep, req.Min, req.Max, req.CPU); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type VPARequest struct {
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Deploy    string `json:"deployment" binding:"required"`
}

func EnsureVPA(c *gin.Context) {
	var req VPARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := service.EnsureVPA(c.Request.Context(), req.Namespace, req.Name, req.Deploy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type BlueGreenRequest struct {
	Namespace     string `json:"namespace" binding:"required"`
	Ingress       string `json:"ingress" binding:"required"`
	Host          string `json:"host"`
	Path          string `json:"path"`
	TargetService string `json:"targetService" binding:"required"`
}

func SwitchBlueGreen(c *gin.Context) {
	var req BlueGreenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := service.SwitchBlueGreen(c.Request.Context(), req.Namespace, req.Ingress, req.Host, req.Path, req.TargetService); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

type HarborGateSyncRequest struct {
	Project    string `json:"project" binding:"required"`
	Repository string `json:"repository" binding:"required"`
	Reference  string `json:"reference" binding:"required"`
	AppName    string `json:"appName" binding:"required"`
	AppNS      string `json:"appNamespace" binding:"required"`
}

// HarborGateSync checks Harbor vulnerability summary and blocks Argo sync if critical>0
func HarborGateSync(c *gin.Context) {
	var req HarborGateSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hc := pkg.NewHarborClientFromEnv()
	rep, err := hc.GetArtifactVulnerabilitySummary(c.Request.Context(), req.Project, req.Repository, req.Reference)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if rep.Critical > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "harbor gate: critical vulnerabilities > 0"})
		return
	}
	if err := pkg.TriggerSync(c.Request.Context(), req.AppName, req.AppNS); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
