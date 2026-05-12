import subprocess
import json
import os
import time

def test_mcp():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    binary = os.path.abspath(os.path.join(script_dir, "../../wisdom_engine.exe"))
    
    if not os.path.exists(binary):
        print(f"[ERROR] Binary not found at {binary}")
        return

    print(f"[INFO] Starting Wisdom MCP Server: {binary}")
    
    # Start process with stdio pipe
    process = subprocess.Popen(
        [binary, "--mcp"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1
    )

    def send_rpc(method, params=None, id=1):
        req = {
            "jsonrpc": "2.0",
            "method": method,
            "params": params or {},
            "id": id
        }
        print(f"[SEND] Sending: {method}")
        process.stdin.write(json.dumps(req) + "\n")
        process.stdin.flush()
        
        line = process.stdout.readline()
        if line:
            resp = json.loads(line)
            print(f"[RECV] Received response for {method}")
            return resp
        return None

    try:
        # 1. Initialize
        init_resp = send_rpc("initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "TestClient", "version": "1.0.0"}
        })
        if init_resp and "result" in init_resp:
            print("[SUCCESS] Initialize Success")
            print(f"   Server: {init_resp['result']['serverInfo']['name']} v{init_resp['result']['serverInfo']['version']}")
        else:
            print("[ERROR] Initialize Failed", init_resp)

        # 2. List Tools
        list_resp = send_rpc("tools/list")
        if list_resp and "result" in list_resp:
            tools = [t['name'] for t in list_resp['result']['tools']]
            print(f"[SUCCESS] Tools List Success: {tools}")
        else:
            print("[ERROR] Tools List Failed", list_resp)

        # 3. Test Call (Recall Wisdom - using a dummy query)
        call_resp = send_rpc("tools/call", {
            "name": "recall_wisdom",
            "arguments": {"query": "test query"}
        })
        if call_resp and "result" in call_resp:
            print("[SUCCESS] Tool Call Success (recall_wisdom)")
        else:
            print("[ERROR] Tool Call Failed", call_resp)

    finally:
        process.terminate()
        print("[INFO] Test process terminated.")

if __name__ == "__main__":
    test_mcp()
