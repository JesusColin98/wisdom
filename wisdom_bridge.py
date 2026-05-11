import sys
import json
import requests
import subprocess
import os

# Wisdom MCP Bridge - Production Edition (IAP & Cloud Run compatible)
# Connects Gemini CLI to the Wisdom Ecosystem on GCP.

# Use the direct Cloud Run URL now that ingress is set to all
SERVICE_URL = os.environ.get("WISDOM_SERVICE_URL", "https://wisdom-engine-3yamn3zhlq-uc.a.run.app")

def get_auth_token():
    """
    Fetches an OIDC ID Token. 
    Optimized for personal accounts (no --audiences) and Service Accounts.
    """
    if "localhost" in SERVICE_URL:
        return None
    try:
        token = subprocess.check_output(["gcloud", "auth", "print-identity-token"], 
                                      text=True, stderr=subprocess.DEVNULL).strip()
        return token
    except:
        return None

def call_wisdom(path, data=None, method="POST"):
    url = f"{SERVICE_URL.rstrip('/')}/{path.lstrip('/')}"
    headers = {"Content-Type": "application/json"}
    
    token = get_auth_token()
    if token:
        headers["Authorization"] = f"Bearer {token}"

    try:
        if method == "POST":
            resp = requests.post(url, json=data, headers=headers, timeout=30, verify=False)
        else:
            resp = requests.get(url, headers=headers, timeout=30, verify=False)
            
        if resp.status_code == 200:
            return resp.json()
        else:
            return {"error": f"Backend HTTP {resp.status_code}", "detail": resp.text[:200]}
    except Exception as e:
        return {"error": f"Connection failed: {str(e)}"}

if __name__ == "__main__":
    for line in sys.stdin:
        try:
            req = json.loads(line)
            method = req.get("method")
            params = req.get("params", {})
            id = req.get("id")
            
            result = None
            error = None

            if method == "initialize":
                result = {
                    "protocolVersion": "2024-11-05",
                    "serverInfo": {"name": "Wisdom-Python-Bridge", "version": "1.0.0"},
                    "capabilities": {"tools": {}}
                }
            elif method == "tools/list":
                result = {
                    "tools": [
                        {
                            "name": "chat",
                            "description": "Sends a message to the Wisdom assistant and gets a grounded response.",
                            "inputSchema": {
                                "type": "object",
                                "properties": {
                                    "message": {"type": "string"}
                                },
                                "required": ["message"]
                            }
                        },
                        {
                            "name": "rem",
                            "description": "Triggers a Rapid Epistemic Metabolism (REM) cycle to consolidate session nodes.",
                            "inputSchema": {
                                "type": "object",
                                "properties": {
                                    "session_id": {"type": "string"}
                                }
                            }
                        },
                        {
                            "name": "calculate_risk",
                            "description": "Analyzes system risk for a given entity.",
                            "inputSchema": {
                                "type": "object",
                                "properties": {
                                    "node_id": {"type": "string"},
                                    "depth": {"type": "integer"}
                                },
                                "required": ["node_id"]
                            }
                        }
                    ]
                }
            elif method == "tools/call":
                tool_name = params.get("name")
                tool_args = params.get("arguments", {})
                
                if tool_name == "chat":
                    res = call_wisdom("chat", {"message": tool_args.get("message", "")})
                    if "error" in res:
                        result = {"content": [{"type": "text", "text": f"Error: {res['error']}"}], "isError": True}
                    else:
                        result = {"content": [{"type": "text", "text": json.dumps(res)}]}
                elif tool_name == "rem":
                    sid = tool_args.get("session_id", "anonymous")
                    res = call_wisdom(f"rem?session_id={sid}")
                    result = {"content": [{"type": "text", "text": json.dumps(res)}]}
                elif tool_name == "calculate_risk":
                    # The api takes node_id as a GET param for handleRisk
                    node_id = tool_args.get("node_id", "")
                    res = call_wisdom(f"cortex/risk?node_id={node_id}", method="GET")
                    result = {"content": [{"type": "text", "text": json.dumps(res)}]}
                else:
                    error = {"code": -32601, "message": "Tool not found"}
            elif method == "notifications/initialized":
                continue
            else:
                # Legacy support just in case
                if method == "chat":
                    result = call_wisdom("chat", {"message": params.get("message", "")})
                elif method == "status":
                    result = call_wisdom("health", method="GET")
                else:
                    error = {"code": -32601, "message": "Method not found"}

            response = {"jsonrpc": "2.0", "id": id}
            if error:
                response["error"] = error
            else:
                response["result"] = result
                
            print(json.dumps(response))
            sys.stdout.flush()
        except Exception as e:
            pass
