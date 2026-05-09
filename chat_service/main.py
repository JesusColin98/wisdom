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

def google_search(query: str):
    """
    Simulated Google Search. 
    In production, this could call the Search API or a specialized MCP.
    """
    print(f"SEARCHING GOOGLE FOR: {query}")
    return f"Search results for '{query}': High-performance Cloud Run deployment patterns found."

async def agent_process(message: str, image: Optional[Image.Image] = None):
    """
    Core agent logic using Gemini 2.0 Flash.
    """
    context_from_search = ""
    if "search" in message.lower():
        search_query = message.lower().replace("search", "").strip()
        search_result = google_search(search_query)
        context_from_search = f"\n\nContext from Google Search: {search_result}"

    if model and image:
        # Multimodal: Agent 'sees' and 'reads'
        prompt = f"System: You are Wisdom, an expert SRE agent. Analyze the visual frame and text. {message} {context_from_search}"
        response = await asyncio.to_thread(model.generate_content, [prompt, image])
        agent_text = response.text
    elif model:
        # Text-only
        response = await asyncio.to_thread(model.generate_content, f"{message} {context_from_search}")
        agent_text = response.text
    else:
        agent_text = f"[Agent - No API Key]: {message} {context_from_search}"

    # Cortex Grounding (Wisdom Engine)
    try:
        resp = requests.post(
            f"{WISDOM_ENGINE_URL}/chat", 
            json={"message": agent_text},
            timeout=30
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
                # Update retina
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
    port = int(os.environ.get("PORT", 8081))
    uvicorn.run(app, host="0.0.0.0", port=port)
