# Message History Garbage Collection

## Overview

Pushem includes automatic garbage collection for message history to prevent unlimited database growth. Old messages are automatically deleted based on configurable retention policies.

## How It Works

1. **Background Process**: A goroutine runs in the background performing periodic cleanup
2. **Configurable Retention**: Messages older than a specified number of days are deleted
3. **Scheduled Cleanup**: Runs at configurable intervals (default: every 24 hours)
4. **Startup Delay**: First cleanup runs 1 minute after server start
5. **Logging**: Cleanup operations are logged for monitoring

## Configuration

### Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `MESSAGE_RETENTION_DAYS` | Days to keep messages | 7 | 30 |
| `CLEANUP_INTERVAL_HOURS` | Hours between cleanup runs | 24 | 6 |

### Examples

**Keep messages for 30 days, cleanup every 6 hours:**
```bash
docker run \
  -e MESSAGE_RETENTION_DAYS=30 \
  -e CLEANUP_INTERVAL_HOURS=6 \
  pushem
```

**Keep messages for 1 day, cleanup hourly:**
```bash
podman-compose up -d -e MESSAGE_RETENTION_DAYS=1 -e CLEANUP_INTERVAL_HOURS=1
```

**Disable cleanup (keep messages forever):**
```bash
# Set retention to a very high value
docker run -e MESSAGE_RETENTION_DAYS=36500 pushem  # ~100 years
```

### Docker Compose

Edit `docker-compose.yml`:

```yaml
services:
  pushem:
    environment:
      - MESSAGE_RETENTION_DAYS=14  # 2 weeks
      - CLEANUP_INTERVAL_HOURS=12  # Twice daily
```

## Monitoring

### Log Output

The server logs cleanup operations:

```
2026/01/28 01:00:00 Message cleanup: retention=7 days, interval=24h0m0s
2026/01/28 02:00:00 Cleaned up 42 old messages (older than 7 days)
2026/01/28 02:00:00 Current message count: 158
```

### Manual Cleanup

You can also manually clear history via the API:

**Clear all messages for a topic:**
```bash
curl -X DELETE http://localhost:8080/history/my-topic
```

**With authentication:**
```bash
curl -X DELETE http://localhost:8080/history/my-topic \
  -H "X-Pushem-Key: your-secret-key"
```

## Database Impact

### Storage Estimates

Approximate storage per message:
- Average message: ~200 bytes
- 1,000 messages: ~200 KB
- 10,000 messages: ~2 MB
- 100,000 messages: ~20 MB

### Performance

- **Cleanup Query**: Efficient indexed delete by timestamp
- **Impact**: Minimal - runs during low-traffic periods
- **Lock Duration**: Very brief, doesn't block other operations

### Vacuum

SQLite benefits from occasional VACUUM operations to reclaim space:

```bash
# Connect to the database
sqlite3 data/pushem.db

# Check database size
.dbinfo

# Reclaim deleted space
VACUUM;

# Exit
.quit
```

Or automate with a script:
```bash
#!/bin/bash
sqlite3 data/pushem.db "VACUUM;"
```

## Best Practices

### High-Volume Systems

For systems with many notifications:

1. **Shorter Retention**: Keep 1-3 days instead of 7
2. **Frequent Cleanup**: Run every 6-12 hours
3. **Regular Vacuum**: Schedule weekly VACUUM operations

```yaml
environment:
  - MESSAGE_RETENTION_DAYS=3
  - CLEANUP_INTERVAL_HOURS=6
```

### Low-Volume Systems

For systems with few notifications:

1. **Longer Retention**: Keep 30-90 days for better history
2. **Infrequent Cleanup**: Daily or weekly is fine

```yaml
environment:
  - MESSAGE_RETENTION_DAYS=30
  - CLEANUP_INTERVAL_HOURS=24
```

### Compliance Requirements

For systems with data retention policies:

1. **Match Requirements**: Set retention to comply with regulations
2. **Document Settings**: Keep records of configuration
3. **Audit Logging**: Monitor cleanup operations

```yaml
environment:
  - MESSAGE_RETENTION_DAYS=90  # GDPR example
  - CLEANUP_INTERVAL_HOURS=24
```

## Troubleshooting

### Cleanup Not Running

Check logs for:
```
Message cleanup: retention=7 days, interval=24h0m0s
```

If missing:
1. Verify environment variables are set correctly
2. Check server has been running for at least 1 minute
3. Look for error messages in logs

### Database Growing

If database continues to grow:

1. **Check Retention**: Ensure `MESSAGE_RETENTION_DAYS` is reasonable
2. **Check Interval**: Ensure cleanup is running frequently enough
3. **Manual Cleanup**: Delete old data manually and VACUUM
4. **Check Volume**: Verify notification volume matches expectations

### Performance Issues

If cleanup causes performance problems:

1. **Increase Interval**: Run less frequently during off-peak hours
2. **Reduce Batch Size**: Consider implementing batch limits
3. **Schedule Wisely**: Use longer intervals (48-72 hours)

## Future Enhancements

Potential improvements for consideration:

- [ ] Configurable cleanup schedule (cron-style)
- [ ] Per-topic retention policies
- [ ] Message archiving before deletion
- [ ] Automatic VACUUM after cleanup
- [ ] Cleanup statistics API endpoint
- [ ] Batch size limits for large cleanups
- [ ] Exponential backoff for failed cleanups

## API Reference

### Database Methods

```go
// Delete messages older than specified days
func (db *DB) DeleteOldMessages(daysOld int) (int64, error)

// Get total message count
func (db *DB) GetMessageCount() (int64, error)

// Clear all messages for a topic
func (db *DB) ClearMessages(topic string) error
```

### Usage Example

```go
import "pushem/internal/db"

database, _ := db.New("pushem.db")

// Delete messages older than 7 days
count, err := database.DeleteOldMessages(7)
if err != nil {
    log.Printf("Cleanup failed: %v", err)
} else {
    log.Printf("Deleted %d old messages", count)
}
```

## Related Documentation

- [README.md](README.md) - Main documentation
- [Docker Compose Configuration](docker-compose.yml)
- [API Documentation](README.md#api-endpoints)
