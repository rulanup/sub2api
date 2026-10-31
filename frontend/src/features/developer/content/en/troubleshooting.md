# Troubleshooting

## The browser reports Failed to fetch

This usually indicates a network, TLS, or CORS rejection. Browsers do not expose the full details of a CORS-blocked response to JavaScript. Check the browser Network panel and the target's Access-Control-Allow-Origin configuration.

## 401 or 403

Confirm that the key uses the Bearer scheme, contains no extra whitespace, and belongs to a group that permits the requested model. Never put keys in URLs.

## Request timeout

Check the target URL, network, gateway load, and upstream latency. Playground applies a request timeout and lets you stop a request.

## Streaming interruption

Check network stability, whether the request was cancelled, and whether the server emitted a structured error event. Keep the Request ID and inspect the redacted report in Debug Center.
