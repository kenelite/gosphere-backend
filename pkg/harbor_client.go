package pkg

import (
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"
)

type HarborClient struct {
    baseURL  string
    username string
    password string
    http     *http.Client
}

func NewHarborClientFromEnv() *HarborClient {
    return &HarborClient{
        baseURL:  os.Getenv("HARBOR_URL"),
        username: os.Getenv("HARBOR_USERNAME"),
        password: os.Getenv("HARBOR_PASSWORD"),
        http:     &http.Client{Timeout: 10 * time.Second},
    }
}

type HarborScanReport struct {
    Critical int `json:"critical"`
    High     int `json:"high"`
    Medium   int `json:"medium"`
    Low      int `json:"low"`
}

// GetArtifactVulnerabilitySummary fetches scan summary for project/repo:tag
func (c *HarborClient) GetArtifactVulnerabilitySummary(ctx context.Context, project, repository, reference string) (*HarborScanReport, error) {
    // GET /api/v2.0/projects/{projectName}/repositories/{repositoryName}/artifacts/{reference}/vulnerabilities/summary
    url := fmt.Sprintf("%s/api/v2.0/projects/%s/repositories/%s/artifacts/%s/vulnerabilities/summary", c.baseURL, project, repository, reference)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil { return nil, err }
    if c.username != "" || c.password != "" {
        token := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
        req.Header.Set("Authorization", "Basic "+token)
    }
    resp, err := c.http.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 { return nil, fmt.Errorf("harbor status %d", resp.StatusCode) }
    var r HarborScanReport
    if err := json.NewDecoder(resp.Body).Decode(&r); err != nil { return nil, err }
    return &r, nil
}

