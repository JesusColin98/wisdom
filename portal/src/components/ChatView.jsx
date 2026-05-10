import React, { useState, useEffect, useRef, useCallback } from 'react';
import { MessageSquare, Send, Bot, User, Sparkles, Database, ArrowRight, Wand2, Video, VideoOff, Mic, MicOff, Radio, Shield, AlertCircle } from 'lucide-react';
import { useWisdom } from '../context/WisdomContext';

const ChatView = ({ onDistill }) => {
  const { AGENT_WS, API_BASE } = useWisdom();
  const [messages, setMessages] = useState([
    { role: 'assistant', content: "Hello! I am Wisdom. I can help you explore your semantic knowledge graph and execute SRE tools. How can I assist you today?", context: [] }
  ]);
  const [input, setInput] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const [isLive, setIsLive] = useState(false);
  const [hasVideo, setHasVideo] = useState(false);
  const [hasAudio, setHasMic] = useState(false);
  const [isConnected, setIsConnected] = useState(false);
  const scrollRef = useRef(null);
  const socketRef = useRef(null);
  const videoRef = useRef(null);
  const streamRef = useRef(null);

  useEffect(() => {
    // Initialize WebSocket
    const socket = new WebSocket(`${AGENT_WS}/ws/chat`);
    
    socket.onopen = () => {
        console.log("Cortex link established");
        setIsConnected(true);
    };
    socket.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'status') {
        if (data.content === 'agent_thinking') setIsTyping(true);
      } else if (data.type === 'message') {
        setMessages(prev => [...prev, { 
          role: data.role, 
          content: data.content,
          context: data.context || []
        }]);
        setIsTyping(false);
      }
    };
    socket.onclose = () => {
        console.log("Cortex link severed");
        setIsConnected(false);
    };
    
    socketRef.current = socket;
    return () => socket.close();
  }, [AGENT_WS]);

  useEffect(() => {
    let frameInterval;
    if (isLive && isConnected) {
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');
      
      frameInterval = setInterval(() => {
        if (videoRef.current && videoRef.current.videoWidth > 0) {
          canvas.width = 160; // Low res for bandwidth
          canvas.height = 120;
          ctx.drawImage(videoRef.current, 0, 0, canvas.width, canvas.height);
          canvas.toBlob((blob) => {
            if (blob && isConnected) {
              socketRef.current.send(blob);
            }
          }, 'image/jpeg', 0.5);
        }
      }, 1000); // 1 FPS for prototype
    }
    return () => clearInterval(frameInterval);
  }, [isLive, isConnected]);


  const toggleLive = async () => {
    if (!isLive) {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
        streamRef.current = stream;
        if (videoRef.current) videoRef.current.srcObject = stream;
        setIsLive(true);
        setHasVideo(true);
        setHasMic(true);
      } catch (err) {
        console.error("Media access denied:", err);
      }
    } else {
      if (streamRef.current) {
        streamRef.current.getTracks().forEach(track => track.stop());
      }
      setIsLive(false);
      setHasVideo(false);
      setHasMic(false);
    }
  };

  const handleSend = (e) => {
    e.preventDefault();
    if (!input.trim() || !socketRef.current) return;

    const userMessage = { role: 'user', content: input };
    setMessages(prev => [...prev, userMessage]);
    
    socketRef.current.send(input);
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
      <div className="p-6 border-b border-gray-800 bg-black/20 flex items-center justify-between">
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
            {isLive ? 'Live Session' : 'Go Live'}
          </button>
        </div>
      </div>

      {/* Main Area: Messages + Video */}
      <div className="flex-1 flex overflow-hidden">
        <div 
          ref={scrollRef}
          className="flex-1 overflow-y-auto p-8 space-y-8 custom-scrollbar"
        >
          {messages.map((msg, idx) => (
            <div key={idx} className={`flex gap-6 ${msg.role === 'user' ? 'justify-end' : ''}`}>
              {msg.role === 'assistant' && (
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center shrink-0">
                  <Bot size={20} className="text-indigo-400" />
                </div>
              )}
              
              <div className={`max-w-[70%] space-y-4 ${msg.role === 'user' ? 'order-1' : ''}`}>
                <div className={`p-5 rounded-2xl border relative group/msg ${
                  msg.role === 'user' 
                    ? 'bg-indigo-600 border-indigo-500 text-white shadow-lg shadow-indigo-500/10' 
                    : 'bg-gray-900/50 border-gray-800'
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
                      {msg.context.map((node, nIdx) => (
                        <div key={nIdx} className="p-3 bg-gray-800/30 border border-gray-700/50 rounded-xl flex items-center justify-between group cursor-pointer hover:border-indigo-500/30 transition-all">
                          <span className="text-[11px] font-bold text-gray-400 group-hover:text-indigo-300 truncate mr-4">{node.id}</span>
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
            <div className="flex gap-6 animate-pulse">
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
          <div className="w-80 border-l border-gray-800 p-6 flex flex-col gap-6 bg-black/40 backdrop-blur-2xl animate-in slide-in-from-right-8 duration-500">
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
                <span className="text-[8px] font-black text-white uppercase tracking-tighter">Live Transmission</span>
              </div>
            </div>

            <div className="space-y-4">
              <div className="flex items-center justify-between p-4 bg-gray-900/50 border border-gray-800 rounded-2xl">
                <span className="text-[10px] font-black text-gray-500 uppercase tracking-widest">Latency</span>
                <span className="text-[10px] font-bold text-green-400 font-mono">42ms</span>
              </div>
              <div className="flex gap-2">
                <button 
                  onClick={() => setHasVideo(!hasVideo)}
                  className={`flex-1 p-3 rounded-xl border transition-all ${hasVideo ? 'bg-gray-800 border-gray-700 text-gray-300' : 'bg-red-500/10 border-red-500/30 text-red-400'}`}
                >
                  {hasVideo ? <Video size={18} className="mx-auto" /> : <VideoOff size={18} className="mx-auto" />}
                </button>
                <button 
                  onClick={() => setHasMic(!hasAudio)}
                  className={`flex-1 p-3 rounded-xl border transition-all ${hasAudio ? 'bg-gray-800 border-gray-700 text-gray-300' : 'bg-red-500/10 border-red-500/30 text-red-400'}`}
                >
                  {hasAudio ? <Mic size={18} className="mx-auto" /> : <MicOff size={18} className="mx-auto" />}
                </button>
              </div>
            </div>

            <div className="mt-auto">
              <div className="p-4 bg-indigo-500/5 border border-indigo-500/20 rounded-2xl">
                <p className="text-[9px] font-bold text-indigo-400/80 leading-relaxed uppercase tracking-tighter">
                  Multimodal data is being streamed to the Cortex for real-time neuro-cognitive mapping.
                </p>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Input */}
      <div className="p-8 bg-black/40 backdrop-blur-xl border-t border-gray-800">
        <form onSubmit={handleSend} className="max-w-4xl mx-auto relative group">
          <input 
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Talk to Wisdom... (Try 'Search for Cloud Run WebSockets')"
            className="w-full bg-gray-900 border border-gray-800 rounded-2xl py-5 px-6 pr-16 focus:outline-none focus:border-indigo-500/50 focus:ring-4 focus:ring-indigo-500/10 transition-all text-sm shadow-2xl"
          />
          <button 
            type="submit"
            className="absolute right-3 top-1/2 -translate-y-1/2 p-3 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl transition-all shadow-xl disabled:opacity-50"
            disabled={!input.trim() || isTyping}
          >
            <Send size={20} />
          </button>
        </form>
      </div>
    </div>
  );
};

export default ChatView;
