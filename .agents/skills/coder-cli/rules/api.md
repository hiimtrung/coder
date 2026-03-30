# Rule: coder-node API Reference

## Chat API

### POST /v1/chat

```json
Request:
{
  "message":    "How should I implement JWT refresh tokens?",
  "session_id": "ses-abc123",   // optional — resume conversation
  "context": {
    "inject_memory": true,
    "inject_skills":  true,
    "memory_limit":  5,
    "skill_limit":   3,
    "extra_system":  "You are reviewing Go code"
  }
}

Response:
{
  "reply":       "For JWT refresh tokens...",
  "session_id":  "ses-abc123",
  "context_used": {
    "memory_hits": ["JWT rotation pattern — stored 2026-01"],
    "skill_hits":  ["golang: error handling"]
  },
  "model":  "qwen3.5:0.8b",
  "tokens": { "prompt": 120, "completion": 340 }
}
```

### POST /v1/chat/stream

Same request body. Response is SSE:

```
data: {"delta":"For","session_id":"ses-abc123","done":false}
data: {"delta":" JWT","session_id":"ses-abc123","done":false}
data: {"delta":"","session_id":"ses-abc123","done":true,"context_used":{...}}
```

### GET /v1/sessions

```json
Response: [{ "id": "ses-abc123", "title": "JWT discussion", "message_count": 12, "updated_at": "..." }]
```

## Review API

### POST /v1/review

```json
Request:
{
  "diff":    "--- a/file.go\n+++ b/file.go\n...",
  "context": "Optional extra context about the change",
  "focus":   "security"   // optional: security | performance | correctness | style
}

Response:
{
  "summary":   "This change adds JWT refresh logic...",
  "strengths": ["Good error handling", "Proper token expiry check"],
  "concerns": [
    {
      "severity":    "HIGH",
      "title":       "Missing token invalidation",
      "description": "Old refresh token is not deleted after rotation",
      "location":    "manager.go:156",
      "suggestion":  "Add repo.DeleteRefreshToken(ctx, oldTokenID)"
    }
  ],
  "suggestions": ["Consider adding an integration test for concurrent rotation"],
  "stats": { "files_changed": 2, "concerns_high": 1, "concerns_medium": 0 }
}
```

## Debug API

### POST /v1/debug

```json
Request:
{
  "error":    "panic: runtime error: nil pointer dereference",
  "context":  "// optional file content or extra context",
  "language": "go"
}

Response:
{
  "root_cause":    "The RotateToken method calls m.repo.UpdateAccessTokenHash() but m.repo is nil",
  "location":      "internal/usecase/auth/manager.go:182-196",
  "suggested_fix": "Add nil check: if m.repo == nil { return ... }",
  "similar_issues": ["nil repo check missing in RegenerateBootstrapToken — fixed 2026-01-15"],
  "confidence":    "HIGH",
  "model":         "qwen3.5:0.8b"
}
```

## Auth (Secure Mode)

### POST /auth/login

```json
Request:  { "token": "bootstrap-token-here" }
Response: { "access_token": "client-token-xxx", "client_id": "cli-abc123" }
```

### POST /auth/token/rotate

```
Headers: Authorization: Bearer <access_token>
Response: { "access_token": "new-token-xxx" }
```

## Health

### GET /health

```json
{ "status": "ok", "secure_mode": false }
```
