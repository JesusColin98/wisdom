/* global AudioContext, AudioWorkletNode */
import React, { useState, useEffect, useRef, useCallback } from 'react';
import { MessageSquare, Send, Bot, User, Sparkles, Database, ArrowRight, Wand2, Mic, MicOff, Radio, Shield, AlertCircle, Paperclip, Brain, Loader2 } from 'lucide-react';
import { useWisdom } from '../context/WisdomContext';
import AuraOrb from './AuraOrb';

const ChatView = ({ onDistill }) => {
  const { AGENT_WS, API_BASE } = useWisdom();
  const [messages, setMessages] = useState([
    { role: 'assistant', content: "Hello! I am Wisdom. I can help you explore your semantic knowledge graph and execute SRE tools. How can I assist you today?", context: [] }
  ]);
  const [input, setInput] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const [isLive, setIsLive] = useState(false);
  const [isListening, setIsListening] = useState(false);
  const [isAiSpeaking, setIsAiSpeaking] = useState(false);
  const [isConnected, setIsConnected] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [isREMLoading, setIsREMLoading] = useState(false);
  
  const scrollRef = useRef(null);
  const socketRef = useRef(null);
  const fileInputRef = useRef(null);
  
  // Audio Refs
  const audioContextRef = useRef(null);
  const playbackContextRef = useRef(null);
  const analyserRef = useRef(null);
  const processorRef = useRef(null);
  const nextStartTimeRef = useRef(0);
  const audioSourcesRef = useRef([]);

  // Auto-scroll
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages, isTyping]);

  const stopAudioPlayback = useCallback(() => {
    audioSourcesRef.current.forEach(source => {
      try { source.stop(); } catch { /* already stopped */ }
    });
    audioSourcesRef.current = [];
    nextStartTimeRef.current = 0;
    setIsAiSpeaking(false);
  }, []);

  const playAudioBuffer = useCallback(async (buffer) => {
    if (!playbackContextRef.current) {
        const AudioContextClass = window.AudioContext || window.webkitAudioContext;
        playbackContextRef.current = new AudioContextClass({ sampleRate: 24000 });
    }
    
    try {
      if (!analyserRef.current) {
        analyserRef.current = playbackContextRef.current.createAnalyser();
        analyserRef.current.fftSize = 256;
        analyserRef.current.connect(playbackContextRef.current.destination);
      }

      if (playbackContextRef.current.state === 'suspended') await playbackContextRef.current.resume();

      const float32Data = new Float32Array(buffer.byteLength / 2);
      const view = new DataView(buffer);
      for (let i = 0; i < float32Data.length; i++) {
        float32Data[i] = view.getInt16(i * 2, true) / 32768.0;
      }

      const audioBuffer = playbackContextRef.current.createBuffer(1, float32Data.length, 24000);
      audioBuffer.getChannelData(0).set(float32Data);
      
      const source = playbackContextRef.current.createBufferSource();
      source.buffer = audioBuffer;
      source.connect(analyserRef.current);

      const now = playbackContextRef.current.currentTime;
      const JITTER_HEADROOM = 0.1;
      if (nextStartTimeRef.current < now) {
        nextStartTimeRef.current = now + JITTER_HEADROOM;
      }

      setIsAiSpeaking(true);
      source.start(nextStartTimeRef.current);
      audioSourcesRef.current.push(source);

      nextStartTimeRef.current += audioBuffer.duration;

      source.onended = () => {
        audioSourcesRef.current = audioSourcesRef.current.filter(s => s !== source);
        if (audioSourcesRef.current.length === 0) {
          setIsAiSpeaking(false);
        }
      };
    } catch (e) { console.error("Playback error", e); }
  }, []);

  useEffect(() => {
    const socket = new WebSocket(`${AGENT_WS}/ws/chat`);
    
    socket.onopen = () => {
        console.log("Cortex link established");
        setIsConnected(true);
    };
    
    socket.onmessage = async (event) => {
      let data = event.data;
      
      // Handle Binary (Audio)
      if (data instanceof Blob) {
          const buffer = await data.arrayBuffer();
          playAudioBuffer(buffer);
          return;
      }

      const json = JSON.parse(data);
      if (json.type === 'status') {
        if (json.content === 'agent_thinking') setIsTyping(true);
      } else if (json.type === 'message') {
        setMessages(prev => [...prev, { 
          role: json.role, 
          content: json.content,
          context: json.context || []
        }]);
        setIsTyping(false);
      } else if (json.type === 'interruption') {
          stopAudioPlayback();
      }
    };
    
    socket.onclose = () => {
        console.log("Cortex link severed");
        setIsConnected(false);
    };
    
    socketRef.current = socket;
    return () => socket.close();
  }, [AGENT_WS, playAudioBuffer, stopAudioPlayback]);

  // Video Streaming (Optional multimodal)
  useEffect(() => {
    let frameInterval;
    if (isLive && isConnected) {
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');
      
      frameInterval = setInterval(() => {
        if (videoRef.current && videoRef.current.videoWidth > 0) {
          canvas.width = 160;
          canvas.height = 120;
          ctx.drawImage(videoRef.current, 0, 0, canvas.width, canvas.height);
          canvas.toBlob((blob) => {
            if (blob && isConnected && socketRef.current.readyState === WebSocket.OPEN) {
              socketRef.current.send(blob);
            }
          }, 'image/jpeg', 0.5);
        }
      }, 1000);
    }
    return () => clearInterval(frameInterval);
  }, [isLive, isConnected]);

  const toggleLive = async () => {
    if (!isLive) {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: false });
        streamRef.current = stream;
        if (videoRef.current) videoRef.current.srcObject = stream;
        setIsLive(true);
      } catch (err) {
        console.error("Media access denied:", err);
      }
    } else {
      if (streamRef.current) {
        streamRef.current.getTracks().forEach(track => track.stop());
      }
      setIsLive(false);
    }
  };

  const toggleMic = async () => {
    if (isListening) {
      processorRef.current?.disconnect();
      processorRef.current = null;
      audioContextRef.current?.close();
      audioContextRef.current = null;
      setIsListening(false);
      return;
    }

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: { sampleRate: 16000, channelCount: 1 } });
      const ctx = new AudioContext({ sampleRate: 16000 });
      audioContextRef.current = ctx;

      await ctx.audioWorklet.addModule('/worklets/pcm-processor.js');
      if (ctx.state === 'suspended') await ctx.resume();

      const source = ctx.createMediaStreamSource(stream);
      const processor = new AudioWorkletNode(ctx, 'pcm-processor');
      processorRef.current = processor;

      processor.port.onmessage = (event) => {
        if (socketRef.current?.readyState === WebSocket.OPEN) {
          socketRef.current.send(event.data); // Send raw PCM
        }
      };

      source.connect(processor);
      processor.connect(ctx.destination);
      setIsListening(true);
    } catch (e) {
      console.error("Mic error", e);
    }
  };

  const handleFileUpload = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    setIsUploading(true);
    const formData = new FormData();
    formData.append('document', file);

    try {
      const res = await fetch(`${API_BASE}/cortex/upload`, {
        method: 'POST',
        body: formData
      });
      if (res.ok) {
        setMessages(prev => [...prev, { 
          role: 'user', 
          content: `Uploaded document: ${file.name}` 
        }, {
          role: 'assistant',
          content: `I've ingested "${file.name}". Its content is now part of the session signals and will be processed during the next REM cycle.`
        }]);
      }
    } catch (err) {
      console.error("Upload failed", err);
    } finally {
      setIsUploading(false);
    }
  };

  const handleTriggerREM = async () => {
    setIsREMLoading(true);
    try {
      const res = await fetch(`${API_BASE}/rem?session_id=anonymous`, {
        method: 'POST'
      });
      if (res.ok) {
        const data = await res.json();
        setMessages(prev => [...prev, {
          role: 'assistant',
          content: `REM Cycle Complete. Consolidated ${data.anchored_nodes} new nodes into the deep Cortex.`
        }]);
      }
    } catch (err) {
      console.error("REM trigger failed", err);
    } finally {
      setIsREMLoading(false);
    }
  };

  const handleSend = (e) => {
    e.preventDefault();
    if (!input.trim() || !socketRef.current) return;

    if (input.trim().toLowerCase() === '/rem') {
      handleTriggerREM();
      setInput('');
      return;
    }

    const userMessage = { role: 'user', content: input };
    setMessages(prev => [...prev, userMessage]);
    
    socketRef.current.send(JSON.stringify({ type: 'message', content: input }));
    setInput('');
  };

  const distillMessage = useCallback((content) => {
    const note = {
        id: `chat-distill-${Math.random().toString(36).substr(2, 9)}`,
        content: `# Distilled Wisdom\n\n> Source: Chat Conversation\n\n${content}`
    };
    onDistill(note);
  }, [onDistill]);

  const verifyMessage = async (content, idx) => {
    try {
        const res = await fetch(`${API_BASE}/validate`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ assertion: content })
        });
        if (res.ok) {
            const data = await res.json();
            setMessages(prev => {
                const next = [...prev];
                next[idx].isValid = data.valid;
                next[idx].validationReason = data.reason;
                return next;
            });
        }
    } catch (e) { console.error("Validation failed:", e); }
  };

  return (
    <div className="flex h-full flex-col bg-[#0d1117] text-gray-200">
      {/* Header */}
      <div className="p-6 border-b border-gray-800 bg-black/20 flex items-center justify-between z-10">
        <div className="flex items-center gap-4">
          <div className="p-2 bg-indigo-500/10 rounded-xl border border-indigo-500/20">
            <MessageSquare className="text-indigo-400" size={24} />
          </div>
          <div>
            <h1 className="text-xl font-black text-white tracking-tight">Real-time Wisdom</h1>
            <p className="text-xs text-gray-500 font-bold uppercase tracking-widest mt-0.5 flex items-center gap-2">
              {isConnected ? (
                <><span className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse" /> Link Active</>
              ) : (
                <><span className="w-1.5 h-1.5 rounded-full bg-red-500" /> Link Severed</>
              )}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <button 
            onClick={toggleLive}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl border transition-all font-black text-[10px] uppercase tracking-widest ${
              isLive ? 'bg-red-500/10 border-red-500/30 text-red-400 shadow-[0_0_20px_rgba(239,68,68,0.1)]' : 'bg-gray-800 border-gray-700 text-gray-400 hover:text-white'
            }`}
          >
            <Radio size={14} className={isLive ? 'animate-pulse' : ''} />
            {isLive ? 'Live Stream' : 'Enable Video'}
          </button>
        </div>
      </div>

      {/* Main Area */}
      <div className="flex-1 flex overflow-hidden relative">
        {/* AuraOrb Backdrop */}
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none opacity-20">
            <AuraOrb analyserRef={analyserRef} isTalking={isAiSpeaking} />
        </div>

        <div 
          ref={scrollRef}
          className="flex-1 overflow-y-auto p-8 space-y-8 custom-scrollbar relative z-1"
        >
          {(messages || []).map((msg, idx) => (
            <div key={idx} className={`flex gap-6 ${msg.role === 'user' ? 'justify-end' : ''}`}>
              {msg.role === 'assistant' && (
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center shrink-0">
                  <Bot size={20} className="text-indigo-400" />
                </div>
              )}
              
              <div className={`max-w-[70%] space-y-4 ${msg.role === 'user' ? 'order-1' : ''}`}>
                <div className={`p-5 rounded-2xl border relative group/msg backdrop-blur-md ${
                  msg.role === 'user' 
                    ? 'bg-indigo-600/80 border-indigo-500 text-white shadow-lg shadow-indigo-500/10' 
                    : 'bg-gray-900/40 border-gray-800'
                }`}>
                  <p className={`text-sm leading-relaxed whitespace-pre-wrap ${msg.isValid === false ? 'underline decoration-red-500 decoration-wavy' : ''}`}>{msg.content}</p>
                  
                  {msg.role === 'assistant' && (
                      <div className="absolute -right-4 -bottom-4 flex gap-2 opacity-0 group-hover/msg:opacity-100 transition-all scale-75 group-hover/msg:scale-100">
                        <button 
                            onClick={() => verifyMessage(msg.content, idx)}
                            className="p-2.5 bg-red-500 hover:bg-red-400 text-white rounded-xl shadow-xl flex items-center gap-2"
                            title="Verify against Cortex"
                        >
                            <Shield size={16} />
                            <span className="text-[10px] font-black uppercase tracking-tighter">Verify</span>
                        </button>
                        <button 
                            onClick={() => distillMessage(msg.content)}
                            className="p-2.5 bg-indigo-500 hover:bg-indigo-400 text-white rounded-xl shadow-xl flex items-center gap-2"
                        >
                            <Wand2 size={16} />
                            <span className="text-[10px] font-black uppercase tracking-tighter">Distill</span>
                        </button>
                      </div>
                  )}
                </div>

                {msg.isValid === false && (
                    <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-xl flex items-center gap-2 text-[10px] font-bold text-red-300 uppercase animate-in slide-in-from-top-2">
                        <AlertCircle size={14} />
                        Potential Hallucination: {msg.validationReason}
                    </div>
                )}

                {msg.context && msg.context.length > 0 && (
                  <div className="space-y-3">
                    <div className="flex items-center gap-2 text-[10px] font-black text-gray-500 uppercase tracking-widest">
                      <Database size={12} />
                      Grounded Context
                    </div>
                    <div className="grid grid-cols-1 gap-2">
                      {(msg.context || []).map((node, nIdx) => (
                        <div key={nIdx} className="p-3 bg-gray-800/30 border border-gray-700/50 rounded-xl flex items-center justify-between group cursor-pointer hover:border-indigo-500/30 transition-all">
                          <span className="text-[11px] font-bold text-gray-400 group-hover:text-indigo-300 truncate mr-4">{node?.id}</span>
                          <ArrowRight size={14} className="text-gray-600 group-hover:text-indigo-400 shrink-0" />
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              {msg.role === 'user' && (
                <div className="w-10 h-10 rounded-xl bg-gray-800 border border-gray-700 flex items-center justify-center shrink-0 order-2">
                  <User size={20} className="text-gray-400" />
                </div>
              )}
            </div>
          ))}
          {isTyping && (
            <div className="flex gap-6 animate-pulse relative z-1">
              <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center shrink-0">
                <Sparkles size={20} className="text-indigo-400" />
              </div>
              <div className="p-5 rounded-2xl bg-gray-900/50 border border-gray-800">
                <div className="flex gap-1">
                  <div className="w-1.5 h-1.5 rounded-full bg-gray-600 animate-bounce" />
                  <div className="w-1.5 h-1.5 rounded-full bg-gray-600 animate-bounce [animation-delay:0.2s]" />
                  <div className="w-1.5 h-1.5 rounded-full bg-gray-600 animate-bounce [animation-delay:0.4s]" />
                </div>
              </div>
            </div>
          )}
        </div>

        {isLive && (
          <div className="w-80 border-l border-gray-800 p-6 flex flex-col gap-6 bg-black/40 backdrop-blur-2xl animate-in slide-in-from-right-8 duration-500 z-10">
            <div className="aspect-video rounded-3xl overflow-hidden border border-white/10 shadow-2xl bg-gray-900 relative group">
              <video 
                ref={videoRef} 
                autoPlay 
                playsInline 
                muted 
                className="w-full h-full object-cover grayscale brightness-110 contrast-125"
              />
              <div className="absolute inset-0 bg-gradient-to-t from-black/60 to-transparent pointer-events-none" />
              <div className="absolute top-4 left-4 flex items-center gap-2 bg-red-500 px-2 py-1 rounded-md shadow-lg shadow-red-500/20">
                <div className="w-1.5 h-1.5 rounded-full bg-white animate-pulse" />
                <span className="text-[8px] font-black text-white uppercase tracking-tighter">Live Sensor Feed</span>
              </div>
            </div>

            <div className="mt-auto">
              <div className="p-4 bg-indigo-500/5 border border-indigo-500/20 rounded-2xl">
                <p className="text-[9px] font-bold text-indigo-400/80 leading-relaxed uppercase tracking-tighter">
                  Multimodal streams are mapped to the semantic substrate in real-time.
                </p>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Input */}
      <div className="p-8 bg-black/40 backdrop-blur-xl border-t border-gray-800 z-10">
        <div className="max-w-4xl mx-auto flex items-center gap-4">
          <button 
            onClick={toggleMic}
            className={`p-5 rounded-2xl border transition-all ${
              isListening ? 'bg-red-500 border-red-400 text-white animate-pulse shadow-[0_0_30px_rgba(239,68,68,0.3)]' : 'bg-gray-900 border-gray-800 text-gray-400 hover:text-white'
            }`}
          >
            {isListening ? <MicOff size={24} /> : <Mic size={24} />}
          </button>

          <div className="flex-1 flex items-center gap-3">
            <input 
              type="file" 
              ref={fileInputRef} 
              onChange={handleFileUpload} 
              className="hidden" 
              accept=".pdf,.doc,.docx,.txt"
            />
            <button 
              onClick={() => fileInputRef.current?.click()}
              disabled={isUploading}
              className={`p-5 rounded-2xl bg-gray-900 border border-gray-800 text-gray-400 hover:text-white transition-all ${isUploading ? 'animate-pulse' : ''}`}
              title="Upload to Cortex (Chat Stream)"
            >
              {isUploading ? <Loader2 size={24} className="animate-spin" /> : <Paperclip size={24} />}
            </button>

            <form onSubmit={handleSend} className="flex-1 relative group">
              <input 
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder={isListening ? "Listening..." : "Talk to Wisdom... (Try '/rem' or upload a file)"}
                className="w-full bg-gray-900 border border-gray-800 rounded-2xl py-5 px-6 pr-16 focus:outline-none focus:border-indigo-500/50 focus:ring-4 focus:ring-indigo-500/10 transition-all text-sm shadow-2xl"
                disabled={isListening}
              />
              <button 
                type="submit"
                className="absolute right-3 top-1/2 -translate-y-1/2 p-3 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl transition-all shadow-xl disabled:opacity-50"
                disabled={!input.trim() || isTyping || isListening}
              >
                <Send size={20} />
              </button>
            </form>

            <button 
              onClick={handleTriggerREM}
              disabled={isREMLoading}
              className={`p-5 rounded-2xl border transition-all ${
                isREMLoading ? 'bg-indigo-500/10 border-indigo-500 text-indigo-400 shadow-[0_0_20px_rgba(99,102,241,0.2)]' : 'bg-gray-900 border-gray-800 text-gray-400 hover:text-indigo-400'
              }`}
              title="Trigger REM Cycle (Consolidate Knowledge)"
            >
              {isREMLoading ? <Loader2 size={24} className="animate-spin" /> : <Brain size={24} />}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ChatView;
