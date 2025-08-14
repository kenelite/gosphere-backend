package pkg

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"
)

// VerifyGitHubSignature validates X-Hub-Signature-256 header using the shared secret
func VerifyGitHubSignature(secret string, body []byte, signatureHeader string) bool {
    if secret == "" || signatureHeader == "" {
        return false
    }
    const prefix = "sha256="
    if !strings.HasPrefix(signatureHeader, prefix) {
        return false
    }
    sigHex := strings.TrimPrefix(signatureHeader, prefix)
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := mac.Sum(nil)
    given, err := hex.DecodeString(sigHex)
    if err != nil {
        return false
    }
    return hmac.Equal(expected, given)
}

// GitHubPushPayload minimal fields needed
type GitHubPushPayload struct {
    Ref        string `json:"ref"`
    Repository struct {
        FullName string `json:"full_name"`
        CloneURL string `json:"clone_url"`
        SSHURL   string `json:"ssh_url"`
    } `json:"repository"`
}

// HandleGitHubWebhook parses event and returns structured info
func HandleGitHubWebhook(r *http.Request) (string, *GitHubPushPayload, []byte, error) {
    event := r.Header.Get("X-GitHub-Event")
    body, err := io.ReadAll(r.Body)
    if err != nil {
        return "", nil, nil, err
    }
    secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
    if !VerifyGitHubSignature(secret, body, r.Header.Get("X-Hub-Signature-256")) {
        return event, nil, body, fmt.Errorf("invalid signature")
    }
    if event == "push" {
        var payload GitHubPushPayload
        if err := json.Unmarshal(body, &payload); err != nil {
            return event, nil, body, err
        }
        return event, &payload, body, nil
    }
    return event, nil, body, nil
}

