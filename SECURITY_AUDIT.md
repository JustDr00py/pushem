# Security Audit Report - Pushem

**Date**: 2026-01-28
**Scope**: Full codebase security review
**Severity Levels**: 🔴 Critical | 🟠 High | 🟡 Medium | 🟢 Low | ℹ️ Info

---

## Executive Summary

Conducted a comprehensive security audit of the Pushem notification service codebase. Identified **6 security issues** requiring attention, ranging from critical timing attack vulnerabilities to informational concerns about resource limits.

**Overall Risk**: 🟡 Medium (with mitigations available)

---

## Findings

### 🔴 CRITICAL: Timing Attack Vulnerability in Authentication

**Location**: `internal/api/handlers.go:70`

**Issue**: The authentication check uses standard string comparison (`!=`) which is vulnerable to timing attacks. An attacker could potentially leak the secret key by measuring response times.

```go
// VULNERABLE CODE
if providedKey != secret {
    http.Error(w, "unauthorized: topic is protected", http.StatusUnauthorized)
    return false
}
```

**Impact**:
- Attackers can perform timing attacks to gradually discover topic secret keys
- Each comparison leaks information about character matches
- With enough samples, secrets can be extracted

**Recommendation**: Use constant-time comparison from `crypto/subtle` package:

```go
import "crypto/subtle"

// SECURE CODE
if subtle.ConstantTimeCompare([]byte(providedKey), []byte(secret)) != 1 {
    http.Error(w, "unauthorized: topic is protected", http.StatusUnauthorized)
    return false
}
```

**Risk Assessment**: HIGH - Exploitable over network, but requires many requests and statistical analysis

---

### 🟠 HIGH: Request Body Size Not Limited (DoS)

**Location**: `internal/api/handlers.go:185`

**Issue**: The `Publish` endpoint uses `io.ReadAll(r.Body)` without size limits, allowing attackers to exhaust memory with large payloads.

```go
// VULNERABLE CODE
body, err := io.ReadAll(r.Body)
```

**Impact**:
- Memory exhaustion DoS attack
- Server crash or slowdown
- Resource exhaustion affecting all users

**Recommendation**: Use `http.MaxBytesReader` to limit request body size:

```go
// SECURE CODE
const MaxBodySize = 10 * 1024 * 1024 // 10 MB
r.Body = http.MaxBytesReader(w, r.Body, MaxBodySize)
body, err := io.ReadAll(r.Body)
if err != nil {
    http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
    return
}
```

**Risk Assessment**: MEDIUM-HIGH - Easy to exploit, but mitigated by reverse proxy limits in production

---

### ✅ FIXED: CORS Configuration Now Secure by Default

**Location**: `cmd/server/main.go:88-95`

**Previous Issue**: CORS allowed all origins with wildcards.

**Status**: ✅ **FIXED** - Now configurable via environment variable with secure defaults

**Implementation**:
```go
// Secure default (localhost only)
allowedOrigins := []string{"http://localhost:*", "https://localhost:*"}
if corsOrigins := os.Getenv("CORS_ORIGINS"); corsOrigins != "" {
    // Parse comma-separated origins from environment
    allowedOrigins = parseOrigins(corsOrigins)
}
```

**Configuration** (via `.env` file):
```bash
# Single domain
CORS_ORIGINS=https://your-domain.com

# Multiple domains
CORS_ORIGINS=https://domain1.com,https://domain2.com

# Public API (use with caution)
CORS_ORIGINS=https://*,http://*
```

**Security Notes**:
- Default is now localhost only (secure by default)
- Requires explicit configuration for production
- Documented in README.md and .env.example
- Docker Compose uses environment variable

**Risk Assessment**: ✅ **RESOLVED** - Secure defaults, configurable for deployment needs

---

### 🟡 MEDIUM: No Concurrency Limits on Push Notifications

**Location**: `internal/api/handlers.go:244-258`

**Issue**: Publishing to a topic with many subscriptions processes all sends synchronously without limits, blocking the request.

```go
// BLOCKS UNTIL ALL SENDS COMPLETE
for _, sub := range subscriptions {
    err := h.webpush.SendNotification(...)
    // ...
}
```

**Impact**:
- Slow response times for topics with many subscribers
- Potential timeout on large subscriber lists
- Server resources tied up during long operations
- Goroutine exhaustion if many publish requests arrive simultaneously

**Recommendation**: Implement concurrent sends with limits:

```go
const MaxConcurrentPushes = 10

sem := make(chan struct{}, MaxConcurrentPushes)
var wg sync.WaitGroup
var mu sync.Mutex

for _, sub := range subscriptions {
    wg.Add(1)
    go func(s db.Subscription) {
        defer wg.Done()
        sem <- struct{}{}        // Acquire
        defer func() { <-sem }() // Release

        err := h.webpush.SendNotification(s.Endpoint, s.P256dh, s.Auth, payload)

        mu.Lock()
        if err != nil {
            failed++
        } else {
            sent++
        }
        mu.Unlock()
    }(sub)
}
wg.Wait()
```

**Risk Assessment**: MEDIUM - Affects performance and reliability, not directly exploitable

---

### 🟢 LOW: Secrets Stored in Plain Text

**Location**: `internal/db/sqlite.go:68-76` (database schema)

**Issue**: Topic secrets are stored in plain text in the database. If the database file is compromised, all secrets are exposed.

```sql
CREATE TABLE IF NOT EXISTS topics (
    topic TEXT PRIMARY KEY,
    secret TEXT NOT NULL,  -- Plain text!
    ...
);
```

**Impact**:
- Database breach exposes all topic secrets
- File system access = full compromise
- Backups contain plain text secrets

**Recommendation**: Hash secrets with bcrypt or argon2:

```go
import "golang.org/x/crypto/bcrypt"

// On protect
hashedSecret, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
db.conn.Exec(query, topic, string(hashedSecret))

// On check
err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(providedKey))
if err != nil {
    return false // Unauthorized
}
```

**Risk Assessment**: LOW - Requires file system access, mitigated by file permissions (0600)

**Note**: Implementing this is a breaking change - existing secrets would need migration

---

### ℹ️ INFO: No Rate Limiting

**Location**: All API endpoints

**Issue**: No built-in rate limiting for any endpoints. Attackers can:
- Brute force topic secrets (combined with timing attack)
- Flood the publish endpoint
- DoS through excessive subscriptions

**Impact**:
- Brute force attacks possible
- API abuse potential
- Resource exhaustion

**Recommendation**:
1. **Production**: Use Caddy rate limiting (already documented in CADDY_SETUP.md)
2. **Application**: Consider adding golang.org/x/time/rate for per-IP limits

**Current Mitigation**: Documentation recommends Caddy rate limiting for production

**Risk Assessment**: INFO - Already documented, requires deployment-time configuration

---

### ℹ️ INFO: Missing Content Security Policy

**Location**: `Caddyfile:23-33` (security headers)

**Issue**: No Content-Security-Policy (CSP) header configured, allowing inline scripts and reducing XSS protection depth.

**Recommendation**: Add CSP header to Caddyfile:

```caddyfile
header {
    Content-Security-Policy "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'"
    # ... other headers ...
}
```

**Risk Assessment**: INFO - Defense-in-depth measure, not critical with current input validation

---

### ℹ️ INFO: Database Query Result Limit

**Location**: `internal/db/sqlite.go:150` (GetMessagesByTopic)

**Issue**: History query hardcodes LIMIT 50, which is reasonable but not configurable. For very active topics, this might be insufficient for debugging.

```sql
LIMIT 50  -- Hardcoded
```

**Recommendation**: Make limit configurable via query parameter:

```go
func (db *DB) GetMessagesByTopic(topic string, limit int) ([]Message, error) {
    if limit <= 0 || limit > 1000 {
        limit = 50 // Default/Max
    }
    query := `... LIMIT ?`
    rows, err := db.conn.Query(query, topic, limit)
}
```

**Risk Assessment**: INFO - Current limit is reasonable for typical usage

---

## Positive Security Findings

✅ **Good practices already implemented**:

1. ✓ All SQL queries use parameterized statements (no SQL injection)
2. ✓ Input validation package with comprehensive checks
3. ✓ SSRF protection on subscription endpoints
4. ✓ UTF-8 validation and null byte detection
5. ✓ Path traversal prevention in topic names
6. ✓ VAPID keys stored with 0600 permissions
7. ✓ Reserved topic names blocked
8. ✓ HTTPS enforcement for subscription endpoints
9. ✓ Proper error handling (no sensitive data in errors)
10. ✓ Automatic cleanup of expired subscriptions (410 Gone)

---

## Recommended Priority

### Immediate (Week 1)
1. 🔴 Fix timing attack in authentication (1-2 hours)
2. 🟠 Add request body size limits (30 minutes)

### Short-term (Week 2-3)
3. 🟡 Review and restrict CORS configuration (1 hour)
4. 🟡 Add concurrency limits to push notifications (2-3 hours)

### Long-term (Future release)
5. 🟢 Consider hashing secrets (breaking change, needs migration path)
6. ℹ️ Add Content-Security-Policy header (30 minutes)
7. ℹ️ Make history limit configurable (optional enhancement)

---

## Security Checklist for Deployment

Before deploying to production:

- [ ] Apply timing attack fix (use crypto/subtle)
- [ ] Add request body size limits
- [ ] Configure restrictive CORS for your domain
- [ ] Set up Caddy rate limiting (see CADDY_SETUP.md)
- [ ] Enable HTTPS (required)
- [ ] Set file permissions: `chmod 600 data/pushem.db data/vapid_keys.json`
- [ ] Configure firewall (ports 80, 443 only)
- [ ] Set strong MESSAGE_RETENTION_DAYS (7 default is good)
- [ ] Monitor logs for suspicious activity
- [ ] Use strong topic secrets (16+ chars, random)

---

## Testing Recommendations

### Security Tests to Add

1. **Timing attack test**: Measure response times for correct vs incorrect secrets
2. **DoS test**: Send large request bodies (100MB+)
3. **CORS test**: Verify origin restrictions
4. **Concurrency test**: Publish to topic with 1000+ subscriptions
5. **Input validation test**: Try path traversal, SQL injection, XSS attempts
6. **SSRF test**: Try subscribing with private IP endpoints

---

## References

- [CWE-208: Observable Timing Discrepancy](https://cwe.mitre.org/data/definitions/208.html)
- [CWE-400: Uncontrolled Resource Consumption](https://cwe.mitre.org/data/definitions/400.html)
- [CWE-942: Overly Permissive CORS Policy](https://cwe.mitre.org/data/definitions/942.html)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)

---

## Audit Metadata

- **Auditor**: Claude Code (Automated Security Analysis)
- **Date**: 2026-01-28
- **Version**: Pushem v1.0 (pre-release)
- **Files Reviewed**: 6 Go files, 1 Caddyfile, 1 docker-compose file
- **Lines of Code**: ~1,200
- **Critical Findings**: 1
- **High Findings**: 1
- **Medium Findings**: 2
- **Low Findings**: 1
- **Informational**: 3
