import React, { useEffect, useRef } from 'react';
import { Bot, User } from 'lucide-react';

/**
 * ChatMessageList — scrollable message thread.
 * Accepts:
 *   messages   Array<{ role: 'user'|'assistant', content: string }>
 */
const ChatMessageList = ({ messages }) => {
  const scrollRef = useRef(null);

  // Auto-scroll to bottom on every new message
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' });
    }
  }, [messages]);

  return (
    <div
      ref={scrollRef}
      className="flex-1 overflow-y-auto px-6 py-5 flex flex-col gap-5"
      style={{ scrollbarWidth: 'thin', scrollbarColor: 'rgba(99,102,241,0.2) transparent' }}
    >
      {messages.map((msg, idx) => (
        <div
          key={idx}
          className={`flex gap-3 items-end ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
          style={{
            animation: 'chatFadeUp 0.25s ease-out both',
            animationDelay: `${Math.min(idx * 0.03, 0.15)}s`,
          }}
        >
          {/* AI avatar */}
          {msg.role === 'assistant' && (
            <div className="w-8 h-8 rounded-xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center shrink-0 mb-0.5">
              <Bot size={16} className="text-indigo-400" />
            </div>
          )}

          <div className="max-w-[78%] flex flex-col gap-1">
            {/* Label tag */}
            {msg.role === 'assistant' && (
              <span className="text-[9px] font-black uppercase tracking-[0.2em] text-indigo-400/50 pl-1">
                Wisdom
              </span>
            )}

            {/* Bubble */}
            <div
              className={`px-5 py-3.5 rounded-2xl text-sm leading-relaxed whitespace-pre-wrap ${
                msg.role === 'user'
                  ? 'bg-indigo-600 text-white font-semibold shadow-lg shadow-indigo-500/10 rounded-br-sm'
                  : 'bg-gray-900/60 border border-gray-800/80 text-gray-200 backdrop-blur-md rounded-bl-sm'
              }`}
            >
              {msg.content}
            </div>
          </div>

          {/* User avatar */}
          {msg.role === 'user' && (
            <div className="w-8 h-8 rounded-xl bg-gray-800 border border-gray-700 flex items-center justify-center shrink-0 mb-0.5">
              <User size={16} className="text-gray-400" />
            </div>
          )}
        </div>
      ))}

      <style>{`
        @keyframes chatFadeUp {
          from { opacity: 0; transform: translateY(8px) scale(0.97); }
          to   { opacity: 1; transform: translateY(0)   scale(1); }
        }
      `}</style>
    </div>
  );
};

export default ChatMessageList;
