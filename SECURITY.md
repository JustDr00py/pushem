# Security Features

Pushem includes comprehensive security measures to protect against common vulnerabilities and attacks.

> **See also**: [SECURITY_AUDIT.md](SECURITY_AUDIT.md) for a complete security audit report with findings and recommendations.

## Security Measures Implemented

Pushem includes multiple layers of security protection:

- **Hashed Secret Storage**: Topic secrets are hashed using bcrypt before storage (defense in depth)
- **Admin Password Hashing**: Admin password hashed with bcrypt, never stored or transmitted in plain text
- **Token-Based Authentication**: JWT tokens for admin panel with configurable expiration
- **Timing Attack Protection**: bcrypt comparison provides constant-time verification
- **DoS Prevention**: Request body size limits (10 MB max)
- **Concurrency Control**: Limited parallel push notifications (max 10 concurrent)
- **CORS Protection**: Configurable origins with secure defaults (localhost only)
- **SQL Injection Protection**: All queries use parameterized statements
- **SSRF Protection**: Validates subscription endpoints, blocks private IPs
- **Path Traversal Protection**: Topic name validation blocks `..` and `//`
- **Input Sanitization**: UTF-8 validation, null byte removal, length limits
- **HTTPS Enforcement**: Required for production (Service Workers requirement)
- **Secure File Permissions**: VAPID keys and database stored with 0600 permissions
- **Content Security Policy**: CSP headers configured in Caddy setup

## Input Validation

All API endpoints validate user input to prevent injection attacks and malicious data:

### Topic Name Validation

Topics are validated with the following rules:
- **Allowed characters**: Letters, numbers, hyphens, underscores, and dots only
- **Maximum length**: 100 characters
- **UTF-8 validation**: Ensures valid character encoding
- **Path traversal prevention**: Blocks `..` and `//` sequences
- **Reserved names**: Prevents use of system-reserved topics (admin, system, api, vapid, health, metrics)

**Example of blocked topics:**
```bash
# Path traversal attempt
curl -X POST http://localhost:8080/publish/../admin  # Blocked

# Invalid characters
curl -X POST http://localhost:8080/publish/topic@name  # Blocked

# Reserved name
curl -X POST http://localhost:8080/publish/admin  # Blocked
```

### Message Content Validation

Notification messages are validated for:
- **Maximum title length**: 256 characters
- **Maximum message length**: 4096 characters
- **UTF-8 validation**: Ensures valid character encoding
- **Null byte detection**: Prevents embedded null bytes that can cause issues
- **Automatic sanitization**: Removes dangerous characters and trims whitespace

### Secret Key Validation

Topic protection secrets are validated for:
- **Minimum length**: 8 characters
- **Maximum length**: 256 characters
- **UTF-8 validation**: Ensures valid character encoding
- **Weak password detection**: Blocks common weak passwords (password, 12345678, qwertyui)

**Example:**
```bash
# Weak secret rejected
curl -X POST http://localhost:8080/protect/my-topic \
  -d '{"secret": "password"}'  # Returns: 400 Bad Request

# Strong secret accepted
curl -X POST http://localhost:8080/protect/my-topic \
  -d '{"secret": "my-super-secure-key-2024!"}'  # Success
```

### Endpoint URL Validation

Subscription endpoints are validated to prevent SSRF (Server-Side Request Forgery) attacks:
- **HTTPS requirement**: All endpoints must use HTTPS
- **Private IP blocking**: Prevents callbacks to private networks
  - Blocks localhost (127.0.0.1, 0.0.0.0)
  - Blocks RFC1918 private ranges (192.168.x.x, 10.x.x.x, 172.16-31.x.x)
- **URL format validation**: Ensures proper URL structure

**Blocked endpoints:**
```bash
# HTTP not allowed
{"endpoint": "http://example.com/push"}  # Blocked

# Localhost blocked
{"endpoint": "https://localhost/push"}  # Blocked

# Private IP blocked
{"endpoint": "https://192.168.1.1/push"}  # Blocked
```

## Authentication

### Topic Protection

Topics can be protected with secret keys. Secrets are **hashed using bcrypt** before being stored in the database, providing defense in depth:

- Database compromises don't reveal the actual secrets
- Only the user knows the original secret
- Bcrypt provides built-in protection against timing attacks
- Industry-standard password hashing best practices applied

**How it works:**

```bash
# Protect a topic
curl -X POST http://localhost:8080/protect/my-topic \
  -H "Content-Type: application/json" \
  -d '{"secret": "your-secret-key"}'

# Publish to protected topic (with header)
curl -X POST http://localhost:8080/publish/my-topic \
  -H "X-Pushem-Key: your-secret-key" \
  -d "Protected message"

# Or with query parameter
curl -X POST "http://localhost:8080/publish/my-topic?key=your-secret-key" \
  -d "Protected message"
```

## HTTPS Enforcement

For production deployments, HTTPS is **required**:
- Service Workers only work on HTTPS (except localhost)
- Web Push requires secure contexts
- Use Caddy for automatic Let's Encrypt certificates (see CADDY_SETUP.md)

## Admin Panel Security

The admin panel at `/admin` uses modern, secure authentication:

### Token-Based Authentication

- **Password Hashing**: Admin password is hashed with bcrypt (never stored in plain text)
- **JWT Tokens**: After successful login, server issues a JWT token
- **One-Time Password**: Password only transmitted once during login (not with every request)
- **Token Expiration**: Tokens expire after configurable duration (default: 60 minutes)
- **Secure Headers**: Tokens sent in standard `Authorization: Bearer` header

### Rate Limiting (Brute-Force Protection)

The admin login endpoint is protected by rate limiting:

- **IP-Based Tracking**: Failed login attempts tracked per IP address
- **Configurable Limits**: Default 5 attempts per 15 minutes
- **Automatic Cleanup**: Old attempts automatically removed from memory
- **429 Status Code**: Returns "Too Many Requests" when limit exceeded
- **Reset on Success**: Successful login clears failed attempts for that IP

This prevents attackers from trying thousands of password combinations.

### Configuration

Set in your `.env` file:

```bash
# Admin password (hashed automatically on startup)
ADMIN_PASSWORD=your-secure-admin-password-here

# Token expiry in minutes (default: 60)
ADMIN_TOKEN_EXPIRY_MINUTES=60

# Rate limiting (brute-force protection)
ADMIN_MAX_LOGIN_ATTEMPTS=5        # Max attempts before blocking
ADMIN_LOGIN_RATE_LIMIT_MINUTES=15  # Time window for rate limiting
```

### Security Benefits

1. **Password only sent once** - Reduces exposure window compared to per-request authentication
2. **Bcrypt hashing** - Password hash stored in memory only, never in database or logs
3. **Token expiration** - Limits damage from token theft
4. **Standard JWT** - Industry-standard, well-tested authentication mechanism
5. **HTTPS required** - Production deployments must use HTTPS to protect tokens in transit

### Best Practices

- Set a strong admin password (16+ characters, mixed case, numbers, symbols)
- Use HTTPS in production (tokens transmitted over secure connection)
- Keep `ADMIN_TOKEN_EXPIRY_MINUTES` reasonable (60-120 minutes)
- Logout when finished to clear session token
- Never share your `.env` file

## VAPID Keys Security

VAPID keys are critical for authentication:
- Generated automatically on first run
- Stored in `vapid_keys.json`
- **Keep this file secure** - losing it invalidates all subscriptions
- Use file permissions to restrict access:
  ```bash
  chmod 600 data/vapid_keys.json
  ```

## CORS Configuration

CORS is properly configured in the server:
- Allows necessary origins for Web Push
- Restricts methods to required endpoints only
- Validates origin headers

## Database Security

SQLite database security measures:
- Uses prepared statements to prevent SQL injection
- Automatic cleanup of old messages (configurable)
- File permissions should be restricted:
  ```bash
  chmod 600 data/pushem.db
  ```

## CORS Configuration

Cross-Origin Resource Sharing (CORS) controls which websites can access your Pushem API.

### Secure Defaults

By default, Pushem only allows requests from `localhost`:
- `http://localhost:*`
- `https://localhost:*`

This is secure for development and prevents unauthorized access.

### Production Configuration

Configure CORS in your `.env` file:

```bash
# Single domain (recommended)
CORS_ORIGINS=https://your-domain.com

# Multiple domains
CORS_ORIGINS=https://app1.com,https://app2.com

# Subdomain wildcard
CORS_ORIGINS=https://*.yourdomain.com

# Public API (allows any origin - use with caution)
CORS_ORIGINS=https://*,http://*
```

### Security Recommendations

1. **Private Deployments**: Use exact domain names
   ```bash
   CORS_ORIGINS=https://your-domain.com
   ```

2. **Multi-Domain**: List all allowed domains explicitly
   ```bash
   CORS_ORIGINS=https://domain1.com,https://domain2.com
   ```

3. **Public API**: Only use wildcards if you intend public access
   - Requires strong topic secret keys
   - Consider rate limiting
   - Monitor for abuse

4. **Never use** wildcards for private/internal deployments

### Testing CORS

Test your CORS configuration:

```bash
# Should succeed (if configured)
curl -H "Origin: https://your-domain.com" \
  http://localhost:8080/vapid-public-key

# Should fail (origin not allowed)
curl -H "Origin: https://evil.com" \
  http://localhost:8080/vapid-public-key
```

## Best Practices

### Production Deployment Checklist

1. **Configure environment variables** - Copy and edit `.env` file
   ```bash
   cp .env.example .env
   nano .env
   ```
   Set:
   - `CORS_ORIGINS` - Your domain(s) to restrict API access
   - `VAPID_SUBJECT` - Your email for push authentication
   - `MESSAGE_RETENTION_DAYS` - Message history retention

2. **Use HTTPS** - Required for Service Workers and Web Push
   ```bash
   # Use Caddy for automatic HTTPS
   docker-compose --profile caddy up -d
   ```

3. **Restrict CORS** - Configure allowed origins
   ```bash
   # In .env file
   CORS_ORIGINS=https://your-domain.com
   ```

4. **Protect sensitive topics** - Use secret keys for important notifications
   ```bash
   curl -X POST http://localhost:8080/protect/alerts \
     -d '{"secret": "strong-random-secret"}'
   ```

5. **Secure file permissions**
   ```bash
   chmod 600 data/vapid_keys.json
   chmod 600 data/pushem.db
   chmod 600 .env
   ```

6. **Configure message retention** - Prevent database bloat
   ```bash
   # In .env file
   MESSAGE_RETENTION_DAYS=7
   ```

7. **Monitor logs** - Watch for suspicious activity
   ```bash
   docker-compose logs -f | grep -i "error\|unauthorized"
   ```

8. **Use strong secrets** - For protected topics
   - Minimum 8 characters (recommended: 16+)
   - Mix of letters, numbers, and symbols
   - Avoid common words or patterns

### Network Security

When exposing Pushem to the internet:

1. **Use a reverse proxy** (Caddy recommended)
   - Automatic HTTPS with Let's Encrypt
   - Rate limiting
   - Request filtering

2. **Consider firewall rules**
   ```bash
   # Example: UFW on Ubuntu
   ufw allow 80/tcp
   ufw allow 443/tcp
   ufw enable
   ```

3. **Monitor access logs**
   ```bash
   # Caddy logs (when using --profile caddy)
   docker-compose logs -f caddy
   ```

## Rate Limiting

Currently, Pushem does not include built-in rate limiting. For production use, consider:

1. **Using Caddy rate limiting**:
   ```
   rate_limit {
       zone publish {
           key {remote_host}
           events 10
           window 1m
       }
   }
   ```

2. **Using a WAF** (Web Application Firewall):
   - Cloudflare
   - AWS WAF
   - ModSecurity

3. **Implementing application-level rate limiting** (future feature)

## Vulnerability Reporting

If you discover a security vulnerability:
1. Do NOT open a public issue
2. Email the maintainers directly
3. Provide detailed information about the vulnerability
4. Allow reasonable time for a fix before public disclosure

## Security Updates

To stay secure:
1. Keep Pushem updated to the latest version
2. Monitor the GitHub repository for security advisories
3. Update dependencies regularly:
   ```bash
   go get -u ./...
   npm update
   ```

## Limitations

Current security limitations to be aware of:
- No built-in rate limiting (use reverse proxy)
- No user account system (topic-based protection only)
- No audit logging (use reverse proxy logs)
- No IP allowlisting/blocklisting (use firewall)

## Related Documentation

- [GARBAGE_COLLECTION.md](GARBAGE_COLLECTION.md) - Message cleanup configuration
- [CADDY_SETUP.md](CADDY_SETUP.md) - Production HTTPS setup
- [README.md](README.md) - General usage and setup
