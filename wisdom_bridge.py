import sys
import json
import requests
import subprocess
import os
import time

# Wisdom MCP Bridge - Production Edition (IAP & Cloud Run compatible)
# Connects Gemini CLI to the Wisdom Ecosystem on GCP.

# Use the Load Balancer IP/Domain with HTTPS
SERVICE_URL = os.environ.get("WISDOM_SERVICE_URL", "https://34-49-82-216.nip.io")

def get_auth_token():
    """
    Fetches an OIDC ID Token. 
    Optimized for personal accounts (no --audiences) and Service Accounts.
    """
    if "localhost" in SERVICE_URL:
        return None
    try:
        # First try: Standard identity token (Works for personal accounts)
        token = subprocess.check_output(["gcloud", "auth", "print-identity-token"], 
                                      text=True, stderr=subprocess.DEVNULL).strip()
        return token
    except:
        try:
            # Second try: With audience (Works for Service Accounts)
            token = subprocess.check_output(["gcloud", "auth", "print-identity-token", f"--audiences={SERVICE_URL}"], 
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
        # For IAP specifically, sometimes you need the token in this header:
        headers["Proxy-Authorization"] = f"Bearer {token}"

    try:
        if method == "POST":
            resp = requests.post(url, json=data, headers=headers, timeout=10, verify=False) # verify=False for nip.io/provisioning
        else:
            resp = requests.get(url, headers=headers, timeout=10, verify=False)
            
        if resp.status_code == 200:
            return resp.json()
        elif resp.status_code == 401 or resp.status_code == 403:
            return {"error": f"Authentication failed (HTTP {resp.status_code}). Please run 'gcloud auth login'"}
        else:
            return {"error": f"Backend returned HTTP {resp.status_code}", "detail": resp.text[:200]}
    except Exception as e:
        return {"error": f"Connection failed: {str(e)}"}

if __name__ == "__main__":
    # Standard MCP JSON-RPC loop
    for line in sys.stdin:
        try:
            req = json.loads(line)
            method = req.get("method")
            params = req.get("params", {})
            id = req.get("id")
            
            if method == "chat":
                result = call_wisdom("chat", {"message": params.get("message", "")})
            elif method == "search":
                result = call_wisdom("chat", {"message": f"search {params.get('query', '')}"})
            elif method == "rem":
                sid = params.get("session_id", "anonymous")
                result = call_wisdom(f"rem?session_id={sid}")
            elif method == "status":
                result = call_wisdom("health", method="GET")
            else:
                result = {"error": "Method not implemented"}

            print(json.dumps({"jsonrpc": "2.0", "result": result, "id": id}))
            sys.stdout.flush()
        except:
            pass
