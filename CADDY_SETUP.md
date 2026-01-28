# Caddy Setup Guide for Pushem

This guide explains how to deploy Pushem with Caddy for automatic HTTPS in production.

## Prerequisites

- A domain name pointing to your server (A record)
- Docker or Podman installed
- Ports 80 and 443 available on your server

## Quick Setup

### 1. Configure Your Domain

Edit the `Caddyfile` and replace the placeholder domain:

```bash
nano Caddyfile
```

Change:
```
pushem.example.com {
    email admin@example.com
    ...
}
```

To your actual domain:
```
push.yourdomain.com {
    email you@yourdomain.com
    ...
}
```

### 2. Start with Caddy

**Option A: Using Docker Compose profiles**
```bash
docker-compose --profile caddy up -d
```

**Option B: Using separate compose file**
```bash
docker-compose -f docker-compose.yml -f docker-compose.caddy.yml up -d
```

**Option C: With Podman**
```bash
podman-compose --profile caddy up -d
```

### 3. Verify Setup

Check that Caddy is running:
```bash
docker-compose logs caddy
```

You should see Caddy obtaining an SSL certificate from Let's Encrypt.

### 4. Access Your Site

Visit `https://your-domain.com` - you should see Pushem with a valid SSL certificate!

## Configuration Options

### Custom Caddyfile

The default Caddyfile includes:
- Automatic HTTPS with Let's Encrypt
- HTTP → HTTPS redirect
- Security headers (HSTS, XSS protection, etc.)
- Gzip compression
- Access logging to `/data/access.log`

### Environment Variables

You can customize Caddy behavior by editing the Caddyfile. Common changes:

**Change log location:**
```caddyfile
log {
    output file /data/custom-access.log
}
```

**Add custom headers:**
```caddyfile
header {
    Custom-Header "value"
}
```

**Enable rate limiting:**
```caddyfile
rate_limit {
    zone static {
        match {
            path /publish/*
        }
        rate 10r/s
    }
}
```

## Troubleshooting

### Certificate Errors

If Caddy can't obtain a certificate:

1. **Check DNS**: Ensure your domain points to your server
   ```bash
   dig your-domain.com
   ```

2. **Check ports**: Ensure ports 80 and 443 are open
   ```bash
   sudo ss -tulpn | grep ':80\|:443'
   ```

3. **Check Caddy logs**:
   ```bash
   docker-compose logs caddy
   ```

### Local Testing

For local testing with self-signed certificates, edit Caddyfile:

```caddyfile
localhost {
    reverse_proxy pushem:8080
    tls internal  # Use internal self-signed cert
}
```

Then access at `https://localhost` (you'll need to accept the self-signed cert warning).

### Firewall Configuration

Make sure your firewall allows traffic on ports 80 and 443:

**UFW (Ubuntu):**
```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 443/udp  # For HTTP/3
```

**Firewalld (RHEL/CentOS):**
```bash
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
```

## Advanced Configuration

### Custom SSL Certificates

If you want to use your own certificates instead of Let's Encrypt:

```caddyfile
your-domain.com {
    tls /path/to/cert.pem /path/to/key.pem
    reverse_proxy pushem:8080
}
```

### Multiple Domains

To serve Pushem on multiple domains:

```caddyfile
push.domain1.com, push.domain2.com {
    reverse_proxy pushem:8080
}
```

### Staging Environment

Use Let's Encrypt staging for testing:

```caddyfile
{
    acme_ca https://acme-staging-v02.api.letsencrypt.org/directory
}

your-domain.com {
    reverse_proxy pushem:8080
}
```

## Monitoring

### View Access Logs

```bash
docker exec -it pushem-caddy-1 tail -f /data/access.log
```

### Check Certificate Expiry

Caddy automatically renews certificates 30 days before expiration. Check status:

```bash
docker exec -it pushem-caddy-1 caddy list-certificates
```

## Backup

Caddy stores certificates in the `caddy-data` volume. To backup:

```bash
docker run --rm \
  -v pushem_caddy-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/caddy-certs-backup.tar.gz /data
```

## Resources

- [Caddy Documentation](https://caddyserver.com/docs/)
- [Caddyfile Syntax](https://caddyserver.com/docs/caddyfile)
- [Let's Encrypt Rate Limits](https://letsencrypt.org/docs/rate-limits/)

## Related Documentation

- [README.md](README.md) - Main documentation
- [SECURITY.md](SECURITY.md) - Security features and best practices
- [GARBAGE_COLLECTION.md](GARBAGE_COLLECTION.md) - Message cleanup configuration
