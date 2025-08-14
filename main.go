package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/kenelite/gosphere-backend/pkg"
	"github.com/kenelite/gosphere-backend/router"
)

func main() {
	// Configure Gin mode
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		gin.SetMode(mode)
	}

	// Initialize dependencies
	if err := pkg.InitDatabase(); err != nil {
		log.Fatalf("failed to init database: %v", err)
	}

	if err := pkg.InitKubernetesClient(); err != nil {
		log.Fatalf("failed to init kubernetes client: %v", err)
	}

	// Initialize Argo CD client based on kube rest config
	if err := pkg.InitArgoClient(); err != nil {
		log.Fatalf("failed to init argo client: %v", err)
	}

	// Optionally init Kafka if brokers provided
	_ = pkg.InitKafka(nil)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) { // minimal request logger
		c.Next()
		status := c.Writer.Status()
		if status >= http.StatusBadRequest {
			log.Printf("%s %s -> %d", c.Request.Method, c.Request.URL.Path, status)
		}
	})

	router.Register(r)

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	if err := r.Run(addr); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
