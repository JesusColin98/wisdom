import os
import json
import asyncio
import io
import base64
import numpy as np
from fastapi import FastAPI, HTTPException, WebSocket, WebSocketDisconnect
from pydantic import BaseModel
from typing import List, Optional
import requests
import uvicorn
from fastapi.middleware.cors import CORSMiddleware
import google.generativeai as genai
from PIL import Image
from pydub import AudioSegment
import speech_recognition as sr
from google.cloud import speech
from google.cloud import texttospeech

app = FastAPI(title="Wisdom Multimodal Agent Service")

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
    # Using Gemini 3.1 Flash Lite for low-latency real-time interaction
    model = genai.GenerativeModel('gemini-3.1-flash-lite')
else:
    print("WARNING: GEMINI_API_KEY not set. Multimodal features will be disabled.")
    model = None

tts_client = texttospeech.TextToSpeechAsyncClient()
stt_client = speech.SpeechAsyncClient()

async def synthesize_speech(text: str) -> bytes:
    try:
        synthesis_input = texttospeech.SynthesisInput(text=text)
        # Using Journey voice for highly realistic, humanized speech
        voice = texttospeech.VoiceSelectionParams(
            language_code="en-US",
            name="en-US-Journey-F"
        )
        audio_config = texttospeech.AudioConfig(
            audio_encoding=texttospeech.AudioEncoding.LINEAR16,
            sample_rate_hertz=24000
        )
        response = await tts_client.synthesize_speech(
            input=synthesis_input, voice=voice, audio_config=audio_config
        )
        return response.audio_content
    except Exception as e:
        print(f"TTS Error: {e}")
        return None

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

async def agent_process(message: str, image: Optional[Image.Image] = None):
    # System Prompt with SRE context
    system_prompt = (
        "You are Wisdom, an expert SRE AI Assistant. "
        "You have access to a semantic knowledge graph (Cortex). "
        "When responding, be technical, concise, and proactive."
    )
    
    contents = [system_prompt]
    if image: contents.append(image)
    contents.append(message)

    if not model:
        return {"response": f"[Offline] {message}", "context_nodes": []}

    try:
        # Generate text response
        response = await asyncio.to_thread(model.generate_content, contents)
        agent_text = response.text

        # Grounding with Wisdom Engine
        headers = {}
        token = get_id_token(WISDOM_ENGINE_URL)
        if token: headers["Authorization"] = f"Bearer {token}"
        
        cortex_resp = requests.post(
            f"{WISDOM_ENGINE_URL}/chat", 
            json={"message": agent_text},
            timeout=10,
            headers=headers
        )
        if cortex_resp.status_code == 200:
            return cortex_resp.json()
        
        return {"response": agent_text, "context_nodes": []}
    except Exception as e:
        return {"response": f"Error: {str(e)}", "context_nodes": []}

@app.websocket("/ws/chat")
async def websocket_endpoint(websocket: WebSocket):
    await websocket.accept()
    print("Multimodal session active")
    
    recognizer = sr.Recognizer()
    audio_queue = asyncio.Queue()
    last_image = None
    
    async def audio_worker():
        """Process incoming audio chunks for transcription."""
        audio_buffer = bytearray()
        silence_timeout = 1.5  # Process after 1.5 seconds of silence
        last_audio_time = asyncio.get_event_loop().time()
        
        while True:
            try:
                chunk = await asyncio.wait_for(audio_queue.get(), timeout=0.5)
                audio_buffer.extend(chunk)
                last_audio_time = asyncio.get_event_loop().time()
                audio_queue.task_done()
            except asyncio.TimeoutError:
                # If we have enough audio and silence timeout reached
                time_since_last = asyncio.get_event_loop().time() - last_audio_time
                if len(audio_buffer) > 16000 * 2 * 1 and time_since_last >= silence_timeout:
                    audio_data = bytes(audio_buffer)
                    audio_buffer.clear()  # Reset for next utterance
                    
                    audio = speech.RecognitionAudio(content=audio_data)
                    config = speech.RecognitionConfig(
                        encoding=speech.RecognitionConfig.AudioEncoding.LINEAR16,
                        sample_rate_hertz=16000,
                        language_code="en-US",
                    )
                    try:
                        print("Transcribing audio...")
                        response = await stt_client.recognize(config=config, audio=audio)
                        transcript = " ".join([r.alternatives[0].transcript for r in response.results]).strip()
                        
                        if transcript:
                            print(f"User (Voice): {transcript}")
                            # Echo user message
                            await websocket.send_json({"type": "message", "role": "user", "content": transcript})
                            await websocket.send_json({"type": "status", "content": "agent_thinking"})
                            
                            result = await agent_process(transcript, last_image)
                            resp_text = result.get("response", "")
                            
                            await websocket.send_json({
                                "type": "message",
                                "role": "assistant",
                                "content": resp_text,
                                "context": result.get("context", [])
                            })
                            
                            audio_bytes = await synthesize_speech(resp_text)
                            if audio_bytes:
                                await websocket.send_bytes(audio_bytes)
                    except Exception as e:
                        print(f"STT Error: {e}")

    worker_task = asyncio.create_task(audio_worker())

    try:
        while True:
            data = await websocket.receive()
            
            if "text" in data:
                try:
                    payload = json.loads(data["text"])
                    if payload.get("type") == "message":
                        message = payload["content"]
                        await websocket.send_json({"type": "status", "content": "agent_thinking"})
                        result = await agent_process(message, last_image)
                        
                        resp_text = result.get("response", "")
                        await websocket.send_json({
                            "type": "message",
                            "role": "assistant",
                            "content": resp_text,
                            "context": result.get("context", [])
                        })
                        
                        # Generate and send TTS audio
                        audio_bytes = await synthesize_speech(resp_text)
                        if audio_bytes:
                            await websocket.send_bytes(audio_bytes)
                except Exception as e:
                    print(f"Error handling text: {e}")
                    # Fallback for raw text
                    await websocket.send_json({"type": "status", "content": "agent_thinking"})
                    result = await agent_process(data["text"], last_image)
                    resp_text = result.get("response", "")
                    await websocket.send_json({"type": "message", "role": "assistant", "content": resp_text})
                    audio_bytes = await synthesize_speech(resp_text)
                    if audio_bytes:
                        await websocket.send_bytes(audio_bytes)
            
            elif "bytes" in data:
                bytes_data = data["bytes"]
                # Heuristic: Small buffers are likely audio (PCM), larger ones are images (JPEG)
                if len(bytes_data) < 10000:
                    await audio_queue.put(bytes_data)
                else:
                    try:
                        last_image = Image.open(io.BytesIO(bytes_data))
                    except: pass

    except WebSocketDisconnect:
        print("Session terminated")
    finally:
        worker_task.cancel()

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8080)
