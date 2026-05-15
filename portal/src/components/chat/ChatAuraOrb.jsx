/* global requestAnimationFrame, cancelAnimationFrame */
import React, { useRef, useEffect } from 'react';

/**
 * ChatAuraOrb — reactive liquid orb that pulses with AI audio frequency data.
 * Ported from ChatBuddy and adapted for the Wisdom portal (indigo theme).
 *
 * Props:
 *   analyserRef   React.RefObject<AnalyserNode>  audio analyser node ref
 *   isTalking     boolean                         true while AI is speaking
 */
const ChatAuraOrb = ({ analyserRef, isTalking }) => {
  const coreRef  = useRef(null);
  const ring1Ref = useRef(null);
  const ring2Ref = useRef(null);

  useEffect(() => {
    const analyser = analyserRef?.current;
    if (!analyser) return;

    const dataArray = new Uint8Array(analyser.frequencyBinCount);
    let animationId;

    const render = () => {
      analyser.getByteFrequencyData(dataArray);

      // Three frequency bands mapped to three visual layers
      let low = 0, mid = 0, high = 0;
      for (let i = 0;  i < 10; i++) low  += dataArray[i];
      for (let i = 10; i < 30; i++) mid  += dataArray[i];
      for (let i = 30; i < 60; i++) high += dataArray[i];

      const lowAvg  = low  / 10 / 255;
      const midAvg  = mid  / 20 / 255;
      const highAvg = high / 30 / 255;

      // Direct DOM mutations — keeps 60 FPS without React re-renders
      if (coreRef.current) {
        const scale = 1 + lowAvg * 0.4;
        coreRef.current.style.transform  = `scale(${scale})`;
        coreRef.current.style.opacity    = `${0.8 + lowAvg * 0.2}`;
        coreRef.current.style.boxShadow  = `0 0 ${20 + lowAvg * 40}px rgba(99,102,241,${0.4 + lowAvg * 0.6})`;
      }
      if (ring1Ref.current) {
        const scale  = 1 + midAvg * 0.7;
        const rotate = midAvg * 30;
        ring1Ref.current.style.transform = `scale(${scale}) rotate(${rotate}deg)`;
        ring1Ref.current.style.opacity   = `${0.3 + midAvg * 0.4}`;
      }
      if (ring2Ref.current) {
        const scale  = 1 + highAvg * 1.1;
        const rotate = -highAvg * 45;
        ring2Ref.current.style.transform = `scale(${scale}) rotate(${rotate}deg)`;
        ring2Ref.current.style.opacity   = `${0.15 + highAvg * 0.4}`;
      }

      animationId = requestAnimationFrame(render);
    };

    if (isTalking) {
      animationId = requestAnimationFrame(render);
    } else {
      // Graceful resting state
      if (coreRef.current) {
        coreRef.current.style.transform = 'scale(1)';
        coreRef.current.style.opacity   = '0.8';
        coreRef.current.style.boxShadow = '0 0 20px rgba(99,102,241,0.3)';
      }
      if (ring1Ref.current) {
        ring1Ref.current.style.transform = 'scale(1) rotate(0deg)';
        ring1Ref.current.style.opacity   = '0.3';
      }
      if (ring2Ref.current) {
        ring2Ref.current.style.transform = 'scale(1) rotate(0deg)';
        ring2Ref.current.style.opacity   = '0.15';
      }
    }

    return () => cancelAnimationFrame(animationId);
  }, [analyserRef, isTalking]);

  return (
    <div className="relative w-72 h-72 flex justify-center items-center">
      <style>{`
        @keyframes wisdomLiquid {
          0%   { border-radius: 40% 60% 70% 30% / 40% 50% 60% 50%; }
          33%  { border-radius: 60% 40% 30% 70% / 60% 30% 70% 40%; }
          66%  { border-radius: 30% 70% 40% 60% / 30% 60% 40% 70%; }
          100% { border-radius: 40% 60% 70% 30% / 40% 50% 60% 50%; }
        }
      `}</style>

      {/* Ambient background glow */}
      <div className="absolute inset-[-40px] bg-[radial-gradient(circle,rgba(99,102,241,0.06)_0%,transparent_70%)] z-0" />

      {/* Ring 2 — outer fluid bloom */}
      <div
        ref={ring2Ref}
        className="absolute w-[220px] h-[220px] bg-gradient-to-br from-indigo-500/40 to-blue-500/40 blur-[35px] z-[1]"
        style={{ opacity: 0.15, animation: 'wisdomLiquid 10s ease-in-out infinite alternate' }}
      />

      {/* Ring 1 — mid reactive energy */}
      <div
        ref={ring1Ref}
        className="absolute w-[160px] h-[160px] bg-gradient-to-tr from-indigo-600/50 to-purple-600/50 blur-[20px] z-[2]"
        style={{ opacity: 0.3, animation: 'wisdomLiquid 7s ease-in-out infinite' }}
      />

      {/* Core — the soul of Wisdom */}
      <div
        ref={coreRef}
        className="absolute w-[100px] h-[100px] bg-[radial-gradient(circle,#ffffff_0%,#c7d2fe_40%,#6366f1_100%)] rounded-full blur-[4px] z-[3]"
        style={{ opacity: 0.8, boxShadow: '0 0 20px rgba(99,102,241,0.3)' }}
      />

      {/* Floor shadow */}
      <div
        className="absolute bottom-[-40px] w-[120px] h-[10px] bg-[radial-gradient(ellipse,rgba(99,102,241,0.2)_0%,transparent_70%)] blur-[5px] scale-x-[2] transition-opacity duration-300"
        style={{ opacity: isTalking ? 0.6 : 0.3 }}
      />
    </div>
  );
};

export default ChatAuraOrb;
