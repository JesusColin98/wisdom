/* global AudioContext, AudioWorkletNode */
import React, { useState, useEffect, useRef, useCallback } from 'react';
import { useWisdom } from '../context/WisdomContext';

// Decoupled chat sub-components
import ChatHeader       from './chat/ChatHeader';
import ChatAuraOrb      from './chat/ChatAuraOrb';
import ChatMessageList  from './chat/ChatMessageList';
import ChatControlPanel from './chat/ChatControlPanel';

/**
 * ChatView — Wisdom Vault Chat.
 *
 * Layout:
 *   ┌─────────────────────────┐
 *   │  ChatHeader             │  fixed
 *   ├─────────────────────────┤
 *   │  AuraOrb viewport       │  50vh (voice) ↕ 80px (text) — CSS transition
 *   ├─────────────────────────┤
 *   │  ChatMessageList        │  flex-1 / scroll
 *   ├─────────────────────────┤
 *   │  ChatControlPanel       │  fixed bottom
 *   └─────────────────────────┘
 *
 * The agent's knowledge source is the Obsidian vault via the
 * `recall_wisdom` tool in chat_service (always called before answering).
 */
const ChatView = () => {
  const { AGENT_WS, activeNamespace } = useWisdom();

  // ── State ──────────────────────────────────────────────────────────────────
  const [messages, setMessages] = useState([
    {
      role: 'assistant',
      content: 'Wisdom online. Cortex linked to your vault. What would you like to explore?',
    },
  ]);
  const [inputText, setInputText]     = useState('');
  const [mode, setMode]               = useState('voice');  // 'voice' | 'text'
  const [isConnected, setIsConnected] = useState(false);
  const [isListening, setIsListening] = useState(false);
  const [isAiSpeaking, setIsAiSpeaking] = useState(false);

  // ── Refs ───────────────────────────────────────────────────────────────────
  const socketRef          = useRef(null);
  const audioContextRef    = useRef(null);   // mic capture context
  const playbackContextRef = useRef(null);   // AI audio playback context
  const analyserRef        = useRef(null);   // feeds AuraOrb
  const processorRef       = useRef(null);   // AudioWorkletNode
  const nextStartTimeRef   = useRef(0);
  const audioSourcesRef    = useRef([]);

  // ── Audio playback ─────────────────────────────────────────────────────────
  const stopAudioPlayback = useCallback(() => {
    audioSourcesRef.current.forEach(s => { try { s.stop(); } catch { /* ok */ } });
    audioSourcesRef.current = [];
    nextStartTimeRef.current = 0;
    setIsAiSpeaking(false);
  }, []);

  const playAudioBuffer = useCallback(async (buffer) => {
    try {
      if (!playbackContextRef.current) {
        const Ctx = window.AudioContext || window.webkitAudioContext;
        playbackContextRef.current = new Ctx({ sampleRate: 24000 });
      }
      const ctx = playbackContextRef.current;
      if (ctx.state === 'suspended') await ctx.resume();

      if (!analyserRef.current) {
        analyserRef.current = ctx.createAnalyser();
        analyserRef.current.fftSize = 256;
        analyserRef.current.connect(ctx.destination);
      }

      const float32 = new Float32Array(buffer.byteLength / 2);
      const view    = new DataView(buffer);
      for (let i = 0; i < float32.length; i++) {
        float32[i] = view.getInt16(i * 2, true) / 32768.0;
      }

      const audioBuf = ctx.createBuffer(1, float32.length, 24000);
      audioBuf.getChannelData(0).set(float32);

      const source = ctx.createBufferSource();
      source.buffer = audioBuf;
      source.connect(analyserRef.current);

      const now = ctx.currentTime;
      const JITTER = 0.15;
      if (nextStartTimeRef.current < now) nextStartTimeRef.current = now + JITTER;

      setIsAiSpeaking(true);
      source.start(nextStartTimeRef.current);
      audioSourcesRef.current.push(source);
      nextStartTimeRef.current += audioBuf.duration;

      source.onended = () => {
        audioSourcesRef.current = audioSourcesRef.current.filter(s => s !== source);
        if (audioSourcesRef.current.length === 0) setIsAiSpeaking(false);
      };
    } catch (e) {
      console.error('[WISDOM-CHAT] Playback error', e);
    }
  }, []);

  // ── WebSocket connection to chat_service ────────────────────────────────────
  useEffect(() => {
    const ws = new WebSocket(`${AGENT_WS}/ws/chat`);

    ws.onopen = () => {
      console.log('[WISDOM-CHAT] Cortex link established');
      setIsConnected(true);
    };

    ws.onmessage = async (event) => {
      const data = event.data;

      // Binary = raw PCM audio from AI
      if (data instanceof Blob) {
        const buf = await data.arrayBuffer();
        playAudioBuffer(buf);
        return;
      }

      try {
        const json = JSON.parse(data);
        if (json.type === 'status') return;
        if (json.type === 'interruption') { stopAudioPlayback(); return; }
        if (json.type === 'message') {
          setMessages(prev => {
            const last = prev[prev.length - 1];
            // Append streamed chunks to the last assistant message
            if (last?.role === 'assistant' && !json.isNew) {
              return [
                ...prev.slice(0, -1),
                { ...last, content: last.content + json.content },
              ];
            }
            return [...prev, { role: json.role || 'assistant', content: json.content }];
          });
        }
      } catch {
        // Ignore malformed frames
      }
    };

    ws.onclose = () => {
      console.log('[WISDOM-CHAT] Cortex link severed');
      setIsConnected(false);
    };

    ws.onerror = (e) => console.error('[WISDOM-CHAT] WS error', e);

    socketRef.current = ws;
    return () => ws.close();
  }, [AGENT_WS, playAudioBuffer, stopAudioPlayback]);

  // ── Mic toggle ─────────────────────────────────────────────────────────────
  const stopMic = useCallback(() => {
    processorRef.current?.disconnect();
    processorRef.current = null;
    audioContextRef.current?.close();
    audioContextRef.current = null;
    setIsListening(false);
  }, []);

  const toggleMic = async () => {
    if (isListening) { stopMic(); return; }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: { sampleRate: 16000, channelCount: 1 },
      });
      const ctx = new AudioContext({ sampleRate: 16000 });
      audioContextRef.current = ctx;

      await ctx.audioWorklet.addModule('/worklets/pcm-processor.js');
      if (ctx.state === 'suspended') await ctx.resume();

      const source    = ctx.createMediaStreamSource(stream);
      const processor = new AudioWorkletNode(ctx, 'pcm-processor');
      processorRef.current = processor;

      processor.port.onmessage = (ev) => {
        if (socketRef.current?.readyState === WebSocket.OPEN) {
          socketRef.current.send(ev.data); // raw PCM bytes
        }
      };

      source.connect(processor);
      processor.connect(ctx.destination);
      setIsListening(true);
    } catch (e) {
      console.error('[WISDOM-CHAT] Mic error', e);
    }
  };

  // ── Mode change — auto-stop mic when leaving voice mode ───────────────────
  const handleModeChange = (newMode) => {
    if (newMode === 'text' && isListening) stopMic();
    setMode(newMode);
  };

  // ── Send text message ──────────────────────────────────────────────────────
  const handleSendText = () => {
    if (!inputText.trim() || socketRef.current?.readyState !== WebSocket.OPEN) return;
    setMessages(prev => [...prev, { role: 'user', content: inputText }]);
    socketRef.current.send(JSON.stringify({ type: 'message', content: inputText }));
    setInputText('');
  };

  // ── Render ──────────────────────────────────────────────────────────────────
  const orbHeight     = mode === 'voice' ? '50vh' : '80px';
  const orbMinHeight  = mode === 'voice' ? '260px' : '80px';
  const orbScale      = mode === 'voice' ? 'scale(1)' : 'scale(0.28)';
  const orbOpacity    = mode === 'voice' ? '1' : '0.4';

  return (
    <div className="flex flex-col h-full bg-[#0d1117] text-gray-100 overflow-hidden relative">
      {/* Ambient background gradients */}
      <div className="absolute inset-0 pointer-events-none overflow-hidden">
        <div
          className="absolute top-0 right-0 w-80 h-80 rounded-full blur-[80px]"
          style={{ background: 'radial-gradient(circle, rgba(99,102,241,0.08) 0%, transparent 70%)' }}
        />
        <div
          className="absolute bottom-0 left-0 w-96 h-96 rounded-full blur-[100px]"
          style={{ background: 'radial-gradient(circle, rgba(139,92,246,0.06) 0%, transparent 70%)' }}
        />
      </div>

      {/* ── Header ── */}
      <ChatHeader isConnected={isConnected} namespace={activeNamespace} />

      {/* ── AuraOrb viewport ── */}
      <div
        className="relative flex items-center justify-center overflow-hidden border-b border-gray-800/40"
        style={{
          height: orbHeight,
          minHeight: orbMinHeight,
          background: 'linear-gradient(to bottom, #0d1117 0%, #111827 50%, #0d1117 100%)',
          transition: 'height 0.45s cubic-bezier(0.34,1.56,0.64,1), min-height 0.45s cubic-bezier(0.34,1.56,0.64,1)',
          zIndex: 5,
        }}
      >
        {/* Orb scales down gracefully in text mode */}
        <div
          style={{
            transform: orbScale,
            opacity: orbOpacity,
            transition: 'transform 0.45s cubic-bezier(0.34,1.56,0.64,1), opacity 0.35s ease',
          }}
        >
          <ChatAuraOrb analyserRef={analyserRef} isTalking={isAiSpeaking} />
        </div>

        {/* Text-mode ambient status pill */}
        {mode === 'text' && (
          <span
            className="absolute text-[10px] font-bold uppercase tracking-[0.2em] pointer-events-none"
            style={{
              color: isAiSpeaking ? 'rgba(129,140,248,0.7)' : 'rgba(99,102,241,0.3)',
              marginLeft: '64px',
              transition: 'color 0.3s',
            }}
          >
            {isAiSpeaking ? 'Speaking...' : 'AI Ready'}
          </span>
        )}

        {/* Waveform bars — voice mode only */}
        {mode === 'voice' && (
          <div
            className="absolute bottom-3 left-0 right-0 h-6 flex justify-center items-end gap-[3px] transition-opacity duration-300"
            style={{ opacity: isAiSpeaking ? 1 : 0 }}
          >
            {Array.from({ length: 28 }, (_, i) => (
              <div
                key={i}
                className="w-[3px] bg-indigo-400 rounded-full"
                style={{
                  height: '4px',
                  boxShadow: '0 0 6px rgba(99,102,241,0.5)',
                  animation: isAiSpeaking
                    ? `waveBar 0.5s ease-in-out ${i * 0.03}s infinite alternate`
                    : 'none',
                }}
              />
            ))}
            <style>{waveStyle}</style>
          </div>
        )}
      </div>

      {/* ── Messages ── */}
      <ChatMessageList messages={messages} />

      {/* ── Control Panel ── */}
      <ChatControlPanel
        mode={mode}
        isListening={isListening}
        inputText={inputText}
        onModeChange={handleModeChange}
        onToggleMic={toggleMic}
        onInputChange={setInputText}
        onSendText={handleSendText}
      />
    </div>
  );
};

export default ChatView;
dom() * 18) + 6}px; }
              }
            `}</style>
          </div>
        )}
      </div>

      {/* ── Messages ── */}
      <ChatMessageList messages={messages} />

      {/* ── Control Panel ── */}
      <ChatControlPanel
        mode={mode}
        isListening={isListening}
        inputText={inputText}
        onModeChange={handleModeChange}
        onToggleMic={toggleMic}
        onInputChange={setInputText}
        onSendText={handleSendText}
      />
    </div>
  );
};

export default ChatView;
