import os
import json
import asyncio
import io
from fastapi import FastAPI, HTTPException, WebSocket, WebSocketDisconnect
from pydantic import BaseModel
from typing import List, Optional
import requests
import uvicorn
from fastapi.middleware.cors import CORSMiddleware
import google.generativeai as genai
from PIL import Image

app = FastAPI(title="Wisdom Chat Agent Service")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

# Configuration
WISDOM_ENGINE_URL = os.environ.get("WISDOM_ENGINE_URL", "http://localhost:8080")
GEMINI_API_KEY = os.environ.get("GEMINI_API_KEY")

if GEMINI_API_KEY:
    genai.configure(api_key=GEMINI_API_KEY)
    # Using Gemini 3.1 Flash for cutting-edge multimodal reasoning
    model = genai.GenerativeModel('gemini-3.1-flash-image-preview')
else:
    print("WARNING: GEMINI_API_KEY not set. Multimodal features will be disabled.")
    model = None

class ChatRequest(BaseModel):
    message: str
    session_id: Optional[str] = "anonymous"

class ChatResponse(BaseModel):
    response: str
    context_nodes: Optional[List[dict]] = []

@app.get("/health")
async def health():
    return {"status": "healthy", "service": "chat_agent"}

def get_id_token(audience: str):
    """
    Fetches OIDC token from metadata server for Cloud Run authentication.
    """
    if "localhost" in audience or "wisdom-engine:" in audience:
        return None
    
    metadata_url = f"http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?audience={audience}"
    headers = {"Metadata-Flavor": "Google"}
    try:
        resp = requests.get(metadata_url, headers=headers, timeout=2)
        if resp.status_code == 200:
            return resp.text
    except:
        pass
    return None

async def handle_mcp_call(tool_name: str, params: dict):
    """
    Executes an MCP-like tool by calling the corresponding Wisdom Engine endpoint.
    """
    endpoint_map = {
        "recall_wisdom": "/cortex/recall",
        "trace_causality": "/cortex/causality",
        "calculate_risk": "/cortex/risk",
        "get_lineage": "/cortex/lineage"
    }

    endpoint = endpoint_map.get(tool_name)
    if not endpoint:
        return {"error": f"Tool {tool_name} not found."}

    try:
        headers = {}
        token = get_id_token(WISDOM_ENGINE_URL)
        if token:
            headers["Authorization"] = f"Bearer {token}"

        if tool_name == "recall_wisdom":
            resp = requests.post(f"{WISDOM_ENGINE_URL}{endpoint}", json={
                "query": params.get("query"),
                "user_id": params.get("user_id", "anonymous"),
                "seeds": params.get("seeds", []),
                "uncertainty": params.get("uncertainty", 0.5)
            }, headers=headers)
        elif tool_name == "trace_causality":
            resp = requests.get(f"{WISDOM_ENGINE_URL}{endpoint}?node_id={params.get('node_id')}", headers=headers)
        elif tool_name == "calculate_risk":
            resp = requests.get(f"{WISDOM_ENGINE_URL}{endpoint}?node_id={params.get('node_id')}", headers=headers)
        elif tool_name == "get_lineage":
            resp = requests.get(f"{WISDOM_ENGINE_URL}{endpoint}?id={params.get('id')}&direction={params.get('direction', 'DOWN')}", headers=headers)
        else:
            resp = requests.get(f"{WISDOM_ENGINE_URL}{endpoint}", headers=headers)

        if resp.status_code == 200:
            return resp.json()
    except Exception as e:
        return {"error": str(e)}
    return {"error": f"Failed to execute tool {tool_name}."}

async def agent_process(message: str, image: Optional[Image.Image] = None):
    """
    Enhanced Neural-Socratic Agent logic with MCP execution.
    """
    # 0. MCP Execution Trigger (Simple keyword parsing for demo)
    if message.startswith("/tool"):
        try:
            parts = message.split(" ", 2)
            tool_name = parts[1]
            params = json.loads(parts[2]) if len(parts) > 2 else {}
            return await handle_mcp_call(tool_name, params)
        except Exception as e:
            return {"response": f"MCP Parse Error: {e}"}

    # 1. Intent Detection (Simplified)
    intent = "GENERAL"
    if "why" in message.lower() or "explain" in message.lower():
        intent = "REASON"
    elif "hierarchy" in message.lower() or "org" in message.lower() or "lineage" in message.lower():
        intent = "HIERARCHY"
    elif "risk" in message.lower() or "safety" in message.lower():
        intent = "RISK"

    # Common headers with OIDC auth
    headers = {}
    token = get_id_token(WISDOM_ENGINE_URL)
    if token:
        headers["Authorization"] = f"Bearer {token}"

    # 2. Pre-Retrieval (Neural-Socratic Expansion)
    grounded_context = ""
    if intent == "REASON":
        try:
            resp = requests.post(f"{WISDOM_ENGINE_URL}/reason", json={"query": message}, timeout=10, headers=headers)
            if resp.status_code == 200:
                data = resp.json()
                grounded_context = f"\n\nEngine Reasoning: {data.get('explanation')}\nContext Nodes: {data.get('nodes')}"
        except: pass
    elif intent == "HIERARCHY":
        try:
            resp = requests.get(f"{WISDOM_ENGINE_URL}/cortex/lineage?id=root&direction=DOWN", timeout=5, headers=headers)
            if resp.status_code == 200:
                grounded_context = f"\n\nDetected Hierarchy: {resp.json()}"
        except: pass

    # 3. LLM Generation
    prompt = f"System: You are Wisdom, an expert SRE agent. {message} {grounded_context}"
    if model and image:
        response = await asyncio.to_thread(model.generate_content, [prompt, image])
        agent_text = response.text
    elif model:
        response = await asyncio.to_thread(model.generate_content, prompt)
        agent_text = response.text
    else:
        agent_text = f"[Agent - Offline Mode]: {message} {grounded_context}"

    # 4. Real-time Logit Validation
    try:
        val_resp = requests.post(f"{WISDOM_ENGINE_URL}/validate", json={"assertion": agent_text}, timeout=5, headers=headers)
        if val_resp.status_code == 200:
            val_data = val_resp.json()
            if not val_data.get("valid"):
                agent_text += f"\n\n[STRICT MODE WARNING]: {val_data.get('reason')}"
    except: pass

    # 5. Cortex Grounding (Wisdom Engine final pass)
    try:
        resp = requests.post(
            f"{WISDOM_ENGINE_URL}/chat", 
            json={"message": agent_text},
            timeout=30,
            headers=headers
        )
        if resp.status_code == 200:
            return resp.json()
    except Exception as e:
        print(f"Cortex connection error: {e}")
        
    return {"response": agent_text, "context_nodes": []}

@app.post("/chat", response_model=ChatResponse)
async def chat_http(request: ChatRequest):
    data = await agent_process(request.message)
    return ChatResponse(
        response=data.get("response", "No response."),
        context_nodes=data.get("context_nodes", [])
    )

@app.websocket("/ws/chat")
async def websocket_endpoint(websocket: WebSocket):
    await websocket.accept()
    print("Real-time link open")
    last_image = None
    try:
        while True:
            data = await websocket.receive()
            
            if "text" in data:
                message = data["text"]
                await websocket.send_json({"type": "status", "content": "agent_thinking"})
                
                result = await agent_process(message, last_image)
                
                await websocket.send_json({
                    "type": "message",
                    "role": "assistant",
                    "content": result.get("response", ""),
                    "context": result.get("context_nodes", [])
                })
            
            elif "bytes" in data:
                try:
                    image_data = data["bytes"]
                    last_image = Image.open(io.BytesIO(image_data))
                except Exception as e:
                    print(f"Failed to process image frame: {e}")

    except WebSocketDisconnect:
        print("Real-time link closed")
    except Exception as e:
        print(f"WS Error: {e}")

if __name__ == "__main__":
    port = int(os.environ.get("PORT", 8080)) # Changed default to 8080 for Cloud Run compatibility
    uvicorn.run(app, host="0.0.0.0", port=port)
