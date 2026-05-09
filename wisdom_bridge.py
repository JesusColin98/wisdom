import sys
import json
import requests
import argparse
import subprocess
import os

# Wisdom MCP Bridge
# Connects Gemini CLI to the high-performance Go engine on Cloud Run.

# Cloud Run Service URL (User should configure this)
SERVICE_URL = os.environ.get("WISDOM_SERVICE_URL", "http://localhost:8080")

def ensure_engine_running():
    """Checks if the local engine is running and starts it if necessary."""
    if "localhost" not in SERVICE_URL:
        return
    
    try:
        resp = requests.get(f"{SERVICE_URL}/health", timeout=2)
        if resp.status_code == 200:
            return
    except:
        pass

    print("Wisdom Engine not detected. Attempting to start locally...", file=sys.stderr)
    try:
        # Check if binary exists
        if os.path.exists("./wisdom/wisdom_engine"):
            # Start from project root
            subprocess.Popen(["nohup", "./wisdom/wisdom_engine"], 
                             stdout=open("wisdom_engine.log", "a"), 
                             stderr=subprocess.STDOUT, 
                             preexec_fn=os.setpgrp)
            # Wait for it to start
            for _ in range(5):
                time.sleep(1)
                try:
                    if requests.get(f"{SERVICE_URL}/health", timeout=1).status_code == 200:
                        print("Wisdom Engine started successfully.", file=sys.stderr)
                        return
                except:
                    continue
        else:
            print("Error: Wisdom Engine binary not found in ./wisdom/wisdom_engine. Please build it first.", file=sys.stderr)
    except Exception as e:
        print(f"Failed to start Wisdom Engine: {e}", file=sys.stderr)

def get_auth_token():
    """Fetches OIDC token for Cloud Run authentication."""
    if "localhost" in SERVICE_URL:
        return None
    try:
        token = subprocess.check_output(["gcloud", "auth", "print-identity-token", f"--audiences={SERVICE_URL}"], text=True).strip()
        return token
    except Exception as e:
        return None

def chat_with_wisdom(message):
    ensure_engine_running()
    try:
        headers = {}
        token = get_auth_token()
        if token:
            headers["Authorization"] = f"Bearer {token}"
            
        resp = requests.post(f"{SERVICE_URL}/chat", json={"message": message}, headers=headers)
        if resp.status_code == 200:
            return resp.json()
        else:
            return {"response": f"Error: Backend returned {resp.status_code}"}
    except Exception as e:
        return {"response": f"Error connecting to Wisdom: {e}"}

def rem_cycle(session_id):
    try:
        headers = {}
        token = get_auth_token()
        if token:
            headers["Authorization"] = f"Bearer {token}"

        resp = requests.post(f"{SERVICE_URL}/rem?session_id={session_id}", headers=headers)
        if resp.status_code == 200:
            return resp.json()
        else:
            return {"error": f"Backend returned {resp.status_code}"}
    except Exception as e:
        return {"error": f"Error connecting to Wisdom: {e}"}

if __name__ == "__main__":
    # If running as an MCP server, we read from stdin
    for line in sys.stdin:
        try:
            req = json.loads(line)
            method = req.get("method")
            params = req.get("params", {})
            
            if method == "chat":
                msg = params.get("message", "")
                result = chat_with_wisdom(msg)
            elif method == "search":
                query = params.get("query", "")
                # Proxy to the Agent Service's search capability
                # We assume agent service is on 8081 locally or accessible via URL
                agent_url = os.environ.get("AGENT_SERVICE_URL", SERVICE_URL.replace("8080", "8081"))
                resp = requests.post(f"{agent_url}/chat", json={"message": f"search {query}"})
                result = resp.json() if resp.status_code == 200 else {"error": "Search failed"}
            elif method == "live_link":
                # Returns the WebSocket URL for the portal to use
                agent_url = os.environ.get("AGENT_SERVICE_URL", SERVICE_URL.replace("8080", "8081"))
                result = {"ws_url": agent_url.replace("http", "ws") + "/ws/chat"}
            elif method == "rem":
                sid = params.get("session_id", "anonymous")
                result = rem_cycle(sid)
            else:
                result = {"error": {"code": -32601, "message": "Method not found"}}

            print(json.dumps({"jsonrpc": "2.0", "result": result, "id": req.get("id")}))
            sys.stdout.flush()
        except Exception as e:
            pass
