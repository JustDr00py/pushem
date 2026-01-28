# Security Features

Pushem includes comprehensive security measures to protect against common vulnerabilities and attacks.

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

Topics can be protected with secret keys:

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

## Best Practices

### Production Deployment Checklist

1. **Use HTTPS** - Required for Service Workers and Web Push
   ```bash
   # Use Caddy for automatic HTTPS
   podman-compose -f docker-compose.caddy.yml up -d
   ```

2. **Protect sensitive topics** - Use secret keys for important notifications
   ```bash
   curl -X POST http://localhost:8080/protect/alerts \
     -d '{"secret": "strong-random-secret"}'
   ```

3. **Secure file permissions**
   ```bash
   chmod 600 data/vapid_keys.json
   chmod 600 data/pushem.db
   ```

4. **Configure message retention** - Prevent database bloat
   ```bash
   docker run -e MESSAGE_RETENTION_DAYS=7 ...
   ```

5. **Monitor logs** - Watch for suspicious activity
   ```bash
   docker-compose logs -f | grep -i "error\|unauthorized"
   ```

6. **Use strong secrets** - For protected topics
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
   # Caddy logs location
   docker-compose -f docker-compose.caddy.yml logs -f caddy
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
