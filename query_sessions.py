import requests
import json
import subprocess
import sys
import threading
import queue
import time

SERVICE_URL = "https://nexusstate-mcp-374889098326.us-central1.run.app"
SERVICE_ACCOUNT = "nexusstate-sa@jesus-mvp.iam.gserviceaccount.com"

def get_token():
    return subprocess.check_output([
        "gcloud", "auth", "print-identity-token",
        f"--impersonate-service-account={SERVICE_ACCOUNT}",
        "--include-email",
        f"--audiences={SERVICE_URL}"
    ]).decode('utf-8').strip()

def query():
    sse_url = f"{SERVICE_URL}/mcp/sse"
    token = get_token()
    headers = {"Authorization": f"Bearer {token}"}
    
    q = queue.Queue()

    def listen_sse(url, headers, q):
        try:
            with requests.get(url, headers=headers, stream=True) as r:
                for line in r.iter_lines():
                    if line:
                        decoded = line.decode('utf-8')
                        if decoded.startswith("data: "):
                            data = decoded[6:].strip()
                            q.put(data)
        except Exception as e:
            print(f"SSE Error: {e}", file=sys.stderr)

    listener = threading.Thread(target=listen_sse, args=(sse_url, headers, q), daemon=True)
    listener.start()

    # 1. Get endpoint
    msg_url_suffix = q.get(timeout=10)
    full_msg_url = f"{SERVICE_URL}/{msg_url_suffix.lstrip('/')}"
    
    # 2. Initialize
    requests.post(full_msg_url, json={
        "jsonrpc": "2.0", "id": 1, "method": "initialize",
        "params": {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "session-query", "version": "1.0"}
        }
    }, headers=headers)
    q.get(timeout=10) # Wait for init response
    
    # 3. Call access_wisdom
    requests.post(full_msg_url, json={
        "jsonrpc": "2.0",
        "id": 100,
        "method": "tools/call",
        "params": {
            "name": "access_wisdom",
            "arguments": {
                "request": {
                    "query": "NexusState architecture",
                    "session_id": "default"
                }
            }
        }
    }, headers=headers)
    
    # Wait longer for tool response
    try:
        tool_resp = q.get(timeout=30)
        return json.loads(tool_resp)
    except queue.Empty:
        return {"error": "Timeout"}

res = query()
print(json.dumps(res, indent=2))
