# Wisdom Troubleshooting & Observability Guide

This guide explains how to diagnose and fix issues in the Wisdom Engine using the built-in observability stack.

## 1. Observability Stack
Wisdom uses structured logging and OTel tracing to provide deep visibility into its internals.

### Structured Logs
Logs are emitted in JSON format to `stdout`. When running in the background, they are typically redirected to `wisdom_engine.log`.
- **Key fields:** `time`, `level`, `msg`, `path` (for HTTP requests), `error` (for failures).

### OTel Traces
Traces are initialized in `pkg/observability/tracer.go`. Every major component (Cortex, Thalamus, API) creates spans.

## 2. Common Issues

### 404 Not Found for Assets
If the UI loads but fails to fetch CSS/JS files:
1.  **Check Routing:** Ensure `RegisterHandlers` in `pkg/api/server.go` handles the `/assets/` prefix.
2.  **Verify Public Symlink:** The engine expects a `./public` directory. In development, this is a symlink to `portal/dist`.
    ```bash
    ls -ld public
    ```
3.  **Logs:** Look for "Serving static asset" entries in the log.
    ```bash
    grep "Serving static asset" wisdom_engine.log
    ```

### MIME Type Errors
`Refused to apply style... because its MIME type ('text/plain') is not supported...`
- **Diagnosis:** This usually means the server returned a 404 page (plain text) instead of the actual CSS file.
- **Fix:** Follow the steps for "404 Not Found" above.

## 3. Operations: Starting & Recovery

If you close your session and need to restart the engine:

### Manual Start
```bash
# From the project root
nohup ./wisdom/wisdom_engine > wisdom_engine.log 2>&1 &
```

### Auto-Recovery
The `wisdom_bridge.py` is configured to automatically attempt a restart if the local engine is not responding on `localhost:8080`. Simply calling a Wisdom MCP tool will trigger this check.

## 4. Backend Health Check
```bash
curl http://localhost:8080/health
```
Expected response: `{"status":"OK"}`

## 5. Metabolism (Performance)
Check token efficiency and TSR:
```bash
curl http://localhost:8080/metabolism
```
