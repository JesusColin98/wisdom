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
        "recall_wisdom": ("/cortex/recall", "POST"),
        "trace_causality": ("/cortex/causality", "GET"),
        "calculate_risk": ("/cortex/risk", "GET"),
        "get_lineage": ("/cortex/lineage", "GET")
    }
    
    entry = endpoint_map.get(tool_name)
    if not entry: return {"error": f"Tool {tool_name} not found."}
    
    endpoint, method = entry
    try:
        headers = {}
        token = get_id_token(WISDOM_ENGINE_URL)
        if token: headers["Authorization"] = f"Bearer {token}"
        
        url = f"{WISDOM_ENGINE_URL}{endpoint}"
        if method == "POST":
            resp = requests.post(url, json=params, headers=headers)
        else:
            # Convert params to query string for GET
            # Special case for lineage mapping keys
            query_params = params
            if tool_name == "get_lineage" and "id" not in params and "node_id" in params:
                query_params = {"id": params["node_id"], "direction": params.get("direction", "UP")}
            elif tool_name != "get_lineage" and "node_id" not in params and "id" in params:
                query_params = {"node_id": params["id"]}
                
            resp = requests.get(url, params=query_params, headers=headers)
            
        if resp.status_code == 200: return resp.json()
        return {"error": f"Engine returned {resp.status_code}: {resp.text}"}
    except Exception as e: return {"error": str(e)}

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
            
            # Send setup message with Tools
            setup_msg = {
                "setup": {
                    "model": "models/gemini-2.0-flash-exp",
                    "systemInstruction": {
                        "parts": [{ "text": "You are Wisdom, a personal knowledge assistant. Your knowledge comes exclusively from the user's Obsidian vault, which is indexed in a semantic graph called Cortex. Before answering any question, always call 'recall_wisdom' to retrieve relevant notes from the vault — never answer from general knowledge alone. Ground your responses in vault content: reference specific note titles when available, highlight connections between ideas, and encourage the user to explore those connections. Speak in a clear, thoughtful, and concise tone. If a topic is not found in the vault, say so explicitly and suggest the user add a note on it." }]
                    },
                    "generationConfig": {
                        "responseModalities": ["AUDIO"]
                    },
                    "tools": [
                        {
                            "functionDeclarations": [
                                {
                                    "name": "recall_wisdom",
                                    "description": "Search the deep knowledge graph (Cortex) for relevant context, historical incidents, or architectural patterns.",
                                    "parameters": {
                                        "type": "object",
                                        "properties": {
                                            "query": {"type": "string", "description": "The search query."}
                                        },
                                        "required": ["query"]
                                    }
                                },
                                {
                                    "name": "trace_causality",
                                    "description": "Trace the root cause or downstream effects of a specific node in the graph.",
                                    "parameters": {
                                        "type": "object",
                                        "properties": {
                                            "node_id": {"type": "string", "description": "The unique ID of the node to trace."}
                                        },
                                        "required": ["node_id"]
                                    }
                                },
                                {
                                    "name": "calculate_risk",
                                    "description": "Calculate the blast radius and risk score for a potential change or failure in a node.",
                                    "parameters": {
                                        "type": "object",
                                        "properties": {
                                            "node_id": {"type": "string", "description": "The unique ID of the node to analyze."}
                                        },
                                        "required": ["node_id"]
                                    }
                                },
                                {
                                    "name": "get_lineage",
                                    "description": "Explore the parent/child hierarchy of a node (e.g., service -> pod -> container).",
                                    "parameters": {
                                        "type": "object",
                                        "properties": {
                                            "id": {"type": "string", "description": "The unique ID of the node."},
                                            "direction": {"type": "string", "enum": ["UP", "DOWN"], "description": "Search up for parents or down for children."}
                                        },
                                        "required": ["id", "direction"]
                                    }
                                }
                            ]
                        }
                    ]
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
                            raw_bytes = data["bytes"]
                            # Detect mime type. JPEG starts with FF D8 FF
                            is_jpeg = len(raw_bytes) > 3 and raw_bytes[0] == 0xff and raw_bytes[1] == 0xd8 and raw_bytes[2] == 0xff
                            mime = "image/jpeg" if is_jpeg else "audio/pcm;rate=16000"
                            
                            encoded = base64.b64encode(raw_bytes).decode("utf-8")
                            media_msg = {
                                "realtimeInput": {
                                    "mediaChunks": [{
                                        "mimeType": mime,
                                        "data": encoded
                                    }]
                                }
                            }
                            await gemini_ws.send(json.dumps(media_msg))
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
                            
                            # Handle Tool Calls
                            if "toolCall" in json_data:
                                calls = json_data["toolCall"].get("functionCalls", [])
                                responses = []
                                for call in calls:
                                    print(f"[WISDOM-CHAT] Tool Call: {call['name']}")
                                    result = await handle_mcp_call(call["name"], call["args"])
                                    responses.append({
                                        "name": call["name"],
                                        "id": call["id"],
                                        "response": { "result": result }
                                    })
                                
                                # Send tool response back to Gemini
                                tool_resp_msg = {
                                    "toolResponse": {
                                        "functionResponses": responses
                                    }
                                }
                                await gemini_ws.send(json.dumps(tool_resp_msg))
                                continue

                            if "serverContent" in json_data:
                                content = json_data["serverContent"]
                                if "interrupted" in content:
                                    await websocket.send_json({"type": "interruption"})
                                
                                model_turn = content.get("modelTurn") or content.get("modelDraft")
                                if model_turn and "parts" in model_turn:
                                    for part in model_turn["parts"]:
                                        if "text" in part:
                                            await websocket.send_json({
                                                "type": "message",
                                                "role": "assistant",
                                                "content": part["text"]
                                            })
                                        if "inlineData" in part:
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
            await asyncio.wait([task1, task2], return_when=asyncio.FIRST_COMPLETED)
            task1.cancel(); task2.cancel()
    except Exception as e:
        print(f"[WISDOM-CHAT] Fatal WebSocket Proxy Error: {e}")
        try:
            await websocket.send_json({"type": "error", "message": str(e)})
        except: pass
                
    except Exception as e:
        print(f"[WISDOM-CHAT] Fatal WebSocket Proxy Error: {e}")
        try:
            await websocket.send_json({"type": "error", "message": str(e)})
        except:
            pass

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8080)
