# Errors and rate limits

Sub2API preserves upstream protocol error shapes and adds gateway-level authentication, routing, and rate-limit failures.

## Common statuses

| Status | Meaning | Next step |
| --- | --- | --- |
| 400 | Invalid request shape or parameter | Check JSON, model, and required fields |
| 401 | Authentication failed | Check the Bearer key, expiry, and permissions |
| 403 | Key is not permitted | Check group, model allowlist, and account state |
| 404 | Path or model unavailable | Check version, path, and model name |
| 429 | Rate limited | Reduce concurrency and honor Retry-After when present |
| 5xx | Upstream or gateway failure | Use the Request ID in Debug Center |

## Rate limiting

A 429 may come from a key, group, channel, or upstream provider. Do not raise concurrency to bypass a limit. Honor server-provided wait times and use exponential backoff for transient failures.

## Diagnosis order

1. Inspect status, latency, and the redacted response in Debug Center.
2. Compare the request with API Reference.
3. Confirm that the key's group permits the requested model.
4. For browser Live mode, verify that the target supports CORS.
