# gosphere-backend

Lightweight backend for a Kubernetes-based microservices PaaS.

## Key Features
- Multi-tenancy: Namespace + ResourceQuota; MySQL storage; CRUD APIs.
- Delivery: GitHub Webhook -> Argo CD Application -> Auto deploy.
- Progressive delivery: Canary (weight/header-based) and Blue-Green (switch Ingress backend).
- Autoscaling: HPA (CPU utilization) and VPA (CRD) integration.
- Registry gate: Harbor vulnerability scan as release gate (block if Critical > 0).

## Quick Start
1. Environment variables
   - Database: `MYSQL_USER`, `MYSQL_PASSWORD`, `MYSQL_HOST`, `MYSQL_PORT` (default 3306), `MYSQL_DATABASE`
   - GitHub: `GITHUB_WEBHOOK_SECRET`
   - Harbor: `HARBOR_URL`, `HARBOR_USERNAME`, `HARBOR_PASSWORD`
   - Kafka (optional): `KAFKA_BROKERS=broker1:9092,broker2:9092`
   - Gin: `GIN_MODE=release` (optional), `HTTP_ADDR=:8080` (optional)
   - Kubernetes: in-cluster via InClusterConfig; locally via `KUBECONFIG` or `~/.kube/config`

2. Run
   ```bash
   go build ./...
   ./gosphere-backend
   ```

3. API Routes (partial)
   - Tenants: `POST /api/v1/tenants`, `GET /api/v1/tenants`, `GET /api/v1/tenants/:id`, `DELETE /api/v1/tenants/:id`
   - Webhook: `POST /webhooks/github`
   - Delivery/Release:
     - Canary weight: `POST /api/v1/deploy/canary/weight`
     - Swimlane header: `POST /api/v1/deploy/swimlane/header`
     - Blue-Green switch: `POST /api/v1/deploy/bluegreen/switch`
   - Autoscaling:
     - HPA: `POST /api/v1/autoscaling/hpa`
     - VPA: `POST /api/v1/autoscaling/vpa`
   - Release gate (Harbor + Argo): `POST /api/v1/cicd/harbor-gate-sync`

4. Request Examples
   - Blue-Green switch
     ```json
     {
       "namespace": "default",
       "ingress": "my-app",
       "host": "example.com",
       "path": "/",
       "targetService": "my-app-green"
     }
     ```
   - Harbor gate sync
     ```json
     {
       "project": "demo",
       "repository": "my-app",
       "reference": "v1.2.3",
       "appName": "my-app-main",
       "appNamespace": "argocd"
     }
     ```

## Development & Testing
- Run tests: `go test ./...`
- Key tests:
  - `pkg/github_webhook_test.go`: signature validation and push payload parsing
  - `pkg/harbor_client_test.go`: Harbor vulnerability summary request
  - `service/deploy_service_test.go`: blue-green switch and canary weight

## Notes
- This project uses Argo CD Application CRs; Kubernetes libs v0.29.x. If you change versions, keep dependency alignment.
