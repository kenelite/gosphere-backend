package pkg

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHarborClient_GetArtifactVulnerabilitySummary(t *testing.T) {
	sr := HarborScanReport{Critical: 0, High: 1, Medium: 2, Low: 3}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(sr)
	}))
	defer ts.Close()

	c := &HarborClient{baseURL: ts.URL, http: ts.Client()}
	got, err := c.GetArtifactVulnerabilitySummary(context.Background(), "p", "r", "tag")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Critical != 0 || got.High != 1 {
		t.Fatalf("unexpected: %+v", got)
	}
}
