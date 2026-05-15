import React from 'react';
import { Mic, MicOff, Send, MessageSquare, Radio } from 'lucide-react';

/**
 * ChatControlPanel — bottom bar with mode toggle + voice/text inputs.
 *
 * Props:
 *   mode          'text' | 'voice'
 *   isListening   boolean
 *   inputText     string
 *   onModeChange  (mode: 'text'|'voice') => void
 *   onToggleMic   () => void
 *   onInputChange (value: string) => void
 *   onSendText    () => void
 */
const ChatControlPanel = ({
  mode,
  isListening,
  inputText,
  onModeChange,
  onToggleMic,
  onInputChange,
  onSendText,
}) => {
  const handleKey = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      onSendText();
    }
  };

  return (
    <div
      className="border-t border-gray-800/60 bg-black/50 backdrop-blur-xl z-10"
      style={{ padding: '12px 24px calc(16px + env(safe-area-inset-bottom, 12px)) 24px' }}
    >
      {/* ── Mode Toggle Pill ── */}
      <div className="flex justify-center mb-4">
        <div className="inline-flex gap-0.5 bg-gray-900/80 border border-gray-800/80 rounded-xl p-1">
          {[
            { id: 'text',  label: 'Text',  Icon: MessageSquare },
            { id: 'voice', label: 'Voice', Icon: Radio },
          ].map(({ id, label, Icon }) => (
            <button
              key={id}
              onClick={() => onModeChange(id)}
              className={`flex items-center gap-1.5 px-4 py-2 rounded-lg text-[11px] font-black uppercase tracking-widest transition-all duration-200 border ${
                mode === id
                  ? 'bg-indigo-500/20 border-indigo-500/40 text-indigo-300 shadow-[0_0_12px_rgba(99,102,241,0.15)]'
                  : 'border-transparent text-gray-500 hover:text-gray-300'
              }`}
            >
              <Icon size={11} />
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* ── Voice Mode ── */}
      {mode === 'voice' && (
        <div className="flex flex-col items-center gap-3">
          {/* Hero mic button */}
          <div className="relative flex items-center justify-center">
            {/* Pulsing rings when listening */}
            {isListening && (
              <>
                <span
                  className="absolute w-[88px] h-[88px] rounded-full bg-indigo-500/20 animate-ping"
                  style={{ animationDuration: '1.6s' }}
                />
                <span
                  className="absolute w-[88px] h-[88px] rounded-full bg-indigo-500/10 animate-ping"
                  style={{ animationDuration: '1.6s', animationDelay: '0.4s' }}
                />
              </>
            )}
            <button
              onClick={onToggleMic}
              className={`relative w-[68px] h-[68px] rounded-full flex items-center justify-center border transition-all duration-300 ${
                isListening
                  ? 'bg-gradient-to-br from-indigo-500 to-purple-600 border-transparent shadow-[0_0_28px_rgba(99,102,241,0.55),0_0_60px_rgba(99,102,241,0.2)] text-white'
                  : 'bg-gray-900/80 border-gray-700/60 text-gray-400 hover:border-indigo-500/40 hover:text-indigo-300'
              }`}
            >
              {isListening
                ? <Mic size={28} strokeWidth={2} />
                : <MicOff size={26} strokeWidth={1.5} style={{ opacity: 0.65 }} />
              }
            </button>
          </div>

          {/* Status label */}
          <span
            className="text-[10px] font-bold uppercase tracking-[0.2em] transition-colors duration-300"
            style={{ color: isListening ? 'rgba(129,140,248,0.9)' : 'rgba(255,255,255,0.2)' }}
          >
            {isListening ? 'Listening...' : 'Tap to speak'}
          </span>
        </div>
      )}

      {/* ── Text Mode ── */}
      {mode === 'text' && (
        <div className="flex items-center gap-3 max-w-3xl mx-auto">
          {/* Compact mic toggle — voice still available */}
          <button
            onClick={onToggleMic}
            className={`w-12 h-12 rounded-xl flex items-center justify-center shrink-0 border transition-all ${
              isListening
                ? 'bg-indigo-600 border-indigo-500 text-white shadow-[0_0_16px_rgba(99,102,241,0.4)]'
                : 'bg-gray-900 border-gray-800 text-gray-400 hover:border-indigo-500/30 hover:text-indigo-300'
            }`}
          >
            {isListening ? <Mic size={20} strokeWidth={2} /> : <MicOff size={20} strokeWidth={1.5} />}
          </button>

          {/* Text input */}
          <div className="flex-1 relative">
            <input
              type="text"
              value={inputText}
              onChange={(e) => onInputChange(e.target.value)}
              onKeyDown={handleKey}
              placeholder={isListening ? 'Listening... (voice active)' : 'Ask Wisdom about your vault...'}
              className="w-full bg-gray-900/80 border border-gray-800 rounded-xl py-3.5 pl-5 pr-14 text-sm text-gray-100 placeholder-gray-600 focus:outline-none focus:border-indigo-500/50 focus:ring-2 focus:ring-indigo-500/10 transition-all"
            />
            <button
              onClick={onSendText}
              className={`absolute right-2.5 top-1/2 -translate-y-1/2 w-9 h-9 rounded-lg flex items-center justify-center transition-all ${
                inputText.trim()
                  ? 'bg-indigo-600 text-white hover:bg-indigo-500 shadow-lg'
                  : 'bg-transparent text-gray-600 cursor-default'
              }`}
            >
              <Send size={16} strokeWidth={2.5} />
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default ChatControlPanel;
