import React from 'react';
import { Brain } from 'lucide-react';

/**
 * ChatHeader — connection status bar for the Wisdom Vault Chat.
 * Props:
 *   isConnected  boolean   WebSocket connected state
 *   namespace    string    Active Cortex namespace
 */
const ChatHeader = ({ isConnected, namespace }) => {
  return (
    <div className="flex items-center justify-between px-6 py-4 border-b border-gray-800/60 bg-black/30 backdrop-blur-xl z-10">
      {/* Left: identity */}
      <div className="flex items-center gap-3">
        <div className="p-2 bg-indigo-500/10 rounded-xl border border-indigo-500/20">
          <Brain className="text-indigo-400" size={20} />
        </div>
        <div>
          <h2 className="text-base font-black text-white tracking-tight leading-none">
            Wisdom
          </h2>
          <p className="text-[10px] font-bold text-indigo-400/70 uppercase tracking-[0.15em] mt-0.5">
            Vault Chat
          </p>
        </div>
      </div>

      {/* Right: connection + namespace */}
      <div className="flex items-center gap-4">
        {namespace && (
          <span className="text-[10px] font-black uppercase tracking-widest text-gray-500 bg-gray-800/60 border border-gray-700/50 px-3 py-1.5 rounded-lg">
            {namespace}
          </span>
        )}
        <div className="flex items-center gap-2">
          <span
            className={`w-2 h-2 rounded-full transition-all duration-500 ${
              isConnected
                ? 'bg-green-400 shadow-[0_0_8px_rgba(74,222,128,0.6)] animate-pulse'
                : 'bg-red-500'
            }`}
          />
          <span className="text-[10px] font-bold uppercase tracking-widest text-gray-500">
            {isConnected ? 'Cortex Linked' : 'Disconnected'}
          </span>
        </div>
      </div>
    </div>
  );
};

export default ChatHeader;
