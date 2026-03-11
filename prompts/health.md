# Health Check — DEPRECATED

**Note**: This prompt is no longer used. The agentic health check timer (`agent-health`) has been replaced by the non-agentic `shelley-health-check` script, which runs hourly.

The non-agentic approach ensures health checks work even when the LLM gateway is unavailable. See `bin/shelley-health-check` and `systemd/shelley-health-check.*` for the new implementation.

---

## Legacy Documentation (for reference)

The old health check was meant to run periodically and alert only on problems. It was invoked by Shelley directly, which meant it couldn't detect when Shelley itself was unable to function.

For reference, the old approach would have checked:
- Disk usage (>85%)
- Memory usage (>90%)
- Load average (>2x CPU count)
- Failed systemd units
- Key service status (shelley.service)
