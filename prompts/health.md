# Health Check — runs every 6 hours

You are performing a routine health check. Be quick and quiet. Only act if something is wrong.

## 1. Run diagnostics

Run these commands and capture output:

```bash
# Disk usage — flag if any filesystem is >85%
df -h --output=target,pcent | tail -n +2

# Memory — flag if usage >90%
free -h
free | awk '/^Mem:/ {printf "Memory usage: %.0f%%\n", $3/$2 * 100}'

# Load average — flag if 5-min load > 2x CPU count
nproc
cat /proc/loadavg

# Failed systemd units
systemctl --failed --no-pager --no-legend

# Key services
systemctl is-active shelley.service
```

## 2. Evaluate

Check these conditions:

| Condition | Threshold |
|-----------|-----------|
| Any filesystem usage | > 85% |
| Memory usage | > 90% |
| 5-min load average | > 2× number of CPUs |
| Any systemd unit failed | any |
| shelley.service | not active |

## 3. If everything is normal

Do nothing. No log entry. No email. Just stop.

## 4. If something is wrong

Append a brief entry to `~/.agent/memory/daily/$(date '+%Y-%m-%d').md`:

```
## Health alert — HH:MM

- <what's wrong, one line per issue>
```

Then email the owner:

```bash
curl -s -X POST http://169.254.169.254/gateway/email/send \
  -H "Content-Type: application/json" \
  -d '{"to":"'"$(cat ~/.agent/owner-email)"'","subject":"[Shelley] Health alert – '$(date +%Y-%m-%d %H:%M)'","body":"<what is wrong and current values>"}'
```

The email subject should convey urgency level. The body should include:
- What's wrong
- Current value vs threshold
- Any remediation you already attempted (if applicable)

## 5. Done

No confirmation needed if healthy. Brief confirmation only if an alert was raised.
