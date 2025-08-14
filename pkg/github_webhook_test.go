package pkg

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerifyGitHubSignature(t *testing.T) {
	secret := "s3cr3t"
	body := []byte(`{"ref":"refs/heads/main"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !VerifyGitHubSignature(secret, body, sig) {
		t.Fatal("expected valid signature")
	}
	if VerifyGitHubSignature(secret, body, "sha256=deadbeef") {
		t.Fatal("expected invalid signature")
	}
}

func TestHandleGitHubWebhook_Push(t *testing.T) {
	body := `{"ref":"refs/heads/main","repository":{"full_name":"o/r","clone_url":"https://example.com/r.git"}}`
	req := httptest.NewRequest("POST", "/webhooks/github", strings.NewReader(body))
	secret := "s3cr3t"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", sig)
	t.Setenv("GITHUB_WEBHOOK_SECRET", secret)

	event, payload, raw, err := HandleGitHubWebhook(req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if event != "push" {
		t.Fatalf("unexpected event: %s", event)
	}
	if payload == nil || payload.Ref != "refs/heads/main" {
		t.Fatalf("payload parse failed: %+v", payload)
	}
	if len(raw) == 0 {
		t.Fatal("expected raw body")
	}
}
