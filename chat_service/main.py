import os
import json
import asyncio
import io
import base64
from fastapi import FastAPI, HTTPException, WebSocket, WebSocketDisconnect
from pydantic import BaseModel
from typing import List, Optional
import requests
import uvicorn
from fastapi.middleware.cors import CORSMiddleware
import websockets

app = FastAPI(title="Wisdom Multimodal Agent Service")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

WISDOM_ENGINE_URL = os.environ.get("WISDOM_ENGINE_URL", "http://localhost:8080")
GEMINI_API_KEY = os.environ.get("GEMINI_API_KEY")

class ChatRequest(BaseModel):
    message: str
    session_id: Optional[str] = "anonymous"

class ChatResponse(BaseModel):
    response: str
    context_nodes: Optional[List[dict]] = []

def get_id_token(audience: str):
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
    endpoint_map = {
        "recall_wisdom": "/cortex/recall",
        "trace_causality": "/cortex/causality",
        "calculate_risk": "/cortex/risk",
        "get_lineage": "/cortex/lineage"
    }
    endpoint = endpoint_map.get(tool_name)
    if not endpoint: return {"error": f"Tool {tool_name} not found."}
    try:
        headers = {}
        token = get_id_token(WISDOM_ENGINE_URL)
        if token: headers["Authorization"] = f"Bearer {token}"
        resp = requests.post(f"{WISDOM_ENGINE_URL}{endpoint}", json=params, headers=headers)
        if resp.status_code == 200: return resp.json()
    except Exception as e: return {"error": str(e)}
    return {"error": f"Failed to execute tool {tool_name}."}

@app.websocket("/ws/chat")
async def websocket_endpoint(websocket: WebSocket):
    await websocket.accept()
    print("[WISDOM-CHAT] Multimodal session active")
    
    if not GEMINI_API_KEY:
        await websocket.send_json({"type": "error", "message": "GEMINI_API_KEY not configured on backend."})
        await websocket.close()
        return

    # Use Gemini 2.0 Live API
    gemini_url = f"wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContent?key={GEMINI_API_KEY}"
    
    try:
        async with websockets.connect(gemini_url) as gemini_ws:
            print("[WISDOM-CHAT] Connected to Gemini 2.0 API")
            
            # Send setup message
            setup_msg = {
                "setup": {
                    "model": "models/gemini-2.0-flash-exp",
                    "systemInstruction": {
                        "parts": [{ "text": "You are Wisdom, an expert SRE AI Assistant. You have access to a semantic knowledge graph (Cortex). Be technical, concise, and proactive. Use the tools provided when asked to investigate." }]
                    },
                    "generationConfig": {
                        "responseModalities": ["AUDIO"]
                    }
                }
            }
            await gemini_ws.send(json.dumps(setup_msg))
            
            async def forward_to_gemini():
                try:
                    while True:
                        data = await websocket.receive()
                        if "text" in data:
                            try:
                                payload = json.loads(data["text"])
                                if payload.get("type") == "message":
                                    # Convert to Gemini format
                                    gemini_msg = {
                                        "clientContent": {
                                            "turns": [{ "role": "user", "parts": [{ "text": payload["content"] }] }],
                                            "turnComplete": True
                                        }
                                    }
                                    await gemini_ws.send(json.dumps(gemini_msg))
                            except Exception as e:
                                print(f"[WISDOM-CHAT] Text Error: {e}")
                        elif "bytes" in data:
                            # Frontend sends raw PCM (Int16, 16kHz). Convert to base64.
                            encoded = base64.b64encode(data["bytes"]).decode("utf-8")
                            audio_msg = {
                                "realtimeInput": {
                                    "mediaChunks": [{
                                        "mimeType": "audio/pcm;rate=16000",
                                        "data": encoded
                                    }]
                                }
                            }
                            await gemini_ws.send(json.dumps(audio_msg))
                except WebSocketDisconnect:
                    print("[WISDOM-CHAT] Client disconnected")
                except Exception as e:
                    print(f"[WISDOM-CHAT] Error forwarding to Gemini: {e}")
            
            async def forward_to_client():
                try:
                    while True:
                        msg = await gemini_ws.recv()
                        
                        try:
                            json_data = json.loads(msg)
                            
                            if "setupComplete" in json_data:
                                await websocket.send_json({"type": "status", "content": "agent_thinking"})
                                await websocket.send_json({"type": "message", "role": "assistant", "content": "Wisdom Online. Cortex linked."})
                                continue
                            
                            if "serverContent" in json_data:
                                content = json_data["serverContent"]
                                
                                if "interrupted" in content:
                                    await websocket.send_json({"type": "interruption"})
                                
                                model_turn = content.get("modelTurn") or content.get("modelDraft")
                                if model_turn and "parts" in model_turn:
                                    for part in model_turn["parts"]:
                                        if "text" in part:
                                            # We send text chunks back to frontend
                                            await websocket.send_json({
                                                "type": "message",
                                                "role": "assistant",
                                                "content": part["text"]
                                            })
                                        if "inlineData" in part:
                                            # Audio back to frontend (send binary frame directly after decoding)
                                            audio_b64 = part["inlineData"]["data"]
                                            audio_bytes = base64.b64decode(audio_b64)
                                            await websocket.send_bytes(audio_bytes)
                        except Exception as e:
                            print(f"Error parsing Gemini response: {e}")
                            
                except websockets.exceptions.ConnectionClosed:
                    print("[WISDOM-CHAT] Gemini connection closed")
                except Exception as e:
                    print(f"[WISDOM-CHAT] Error forwarding to client: {e}")
            
            task1 = asyncio.create_task(forward_to_gemini())
            task2 = asyncio.create_task(forward_to_client())
            
            done, pending = await asyncio.wait([task1, task2], return_when=asyncio.FIRST_COMPLETED)
            
            for task in pending:
                task.cancel()
                
    except Exception as e:
        print(f"[WISDOM-CHAT] Fatal WebSocket Proxy Error: {e}")
        try:
            await websocket.send_json({"type": "error", "message": str(e)})
        except:
            pass

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8080)
