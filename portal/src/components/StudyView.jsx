import React, { useState, useEffect, useCallback } from 'react';
import {
  BookOpen, ChevronRight, CheckCircle2, XCircle, Brain,
  Loader2, Sparkles, Trophy, Clock, Target, ArrowRight,
  RotateCcw, Flame, TrendingUp, BarChart2
} from 'lucide-react';
import { useWisdom } from '../context/WisdomContext';

// ─── Grade Buttons ────────────────────────────────────────────────────────────
const GRADES = [
  { value: 1, label: 'Again',  delta: '-0.30', color: 'bg-red-500/20 border-red-500/40 text-red-300 hover:bg-red-500/30',     key: '1' },
  { value: 2, label: 'Hard',   delta: '+0.05', color: 'bg-orange-500/20 border-orange-500/40 text-orange-300 hover:bg-orange-500/30', key: '2' },
  { value: 3, label: 'Good',   delta: '+0.15', color: 'bg-cyan-500/20 border-cyan-500/40 text-cyan-300 hover:bg-cyan-500/30',   key: '3' },
  { value: 4, label: 'Easy',   delta: '+0.30', color: 'bg-emerald-500/20 border-emerald-500/40 text-emerald-300 hover:bg-emerald-500/30', key: '4' },
];

// ─── Mastery Ring ─────────────────────────────────────────────────────────────
function MasteryRing({ score, size = 80 }) {
  const pct = Math.max(0, Math.min(1, score || 0));
  const r = (size / 2) - 8;
  const circ = 2 * Math.PI * r;
  const dash = circ * pct;

  const color = pct < 0.3 ? '#ef4444' : pct < 0.6 ? '#f59e0b' : pct < 0.8 ? '#06b6d4' : '#10b981';
  const label = pct < 0.3 ? 'FRAGILE' : pct < 0.6 ? 'LEARNING' : pct < 0.8 ? 'SOLID' : 'DOMINATED';

  return (
    <div className="flex flex-col items-center gap-1">
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <circle cx={size/2} cy={size/2} r={r} fill="none" stroke="#1f2937" strokeWidth="6" />
        <circle
          cx={size/2} cy={size/2} r={r} fill="none"
          stroke={color} strokeWidth="6"
          strokeDasharray={`${dash} ${circ}`}
          strokeLinecap="round"
          transform={`rotate(-90 ${size/2} ${size/2})`}
          className="transition-all duration-700"
        />
        <text x={size/2} y={size/2} textAnchor="middle" dominantBaseline="middle"
          fill="white" fontSize="13" fontWeight="bold">
          {Math.round(pct * 100)}%
        </text>
      </svg>
      <span className="text-[9px] font-black uppercase tracking-widest" style={{ color }}>{label}</span>
    </div>
  );
}

// ─── Card Display ─────────────────────────────────────────────────────────────
function FlashCard({ card, revealed, onReveal }) {
  const domainColors = {
    CHESS: '#06b6d4', FINANCE: '#10b981', LANGUAGE: '#f59e0b', TECH: '#6366f1', GENERAL: '#8b5cf6',
  };
  const color = domainColors[card.domain] || '#6366f1';

  return (
    <div className="relative w-full max-w-2xl mx-auto">
      {/* Card */}
      <div className="relative min-h-64 p-8 rounded-3xl border-2 border-gray-800/60 bg-gray-900/60 backdrop-blur overflow-hidden cursor-pointer transition-all duration-300 hover:border-gray-700/80"
        onClick={!revealed ? onReveal : undefined}
        style={{ boxShadow: `0 0 40px ${color}10` }}
      >
        {/* Domain stripe */}
        <div className="absolute top-0 left-0 right-0 h-1 rounded-t-3xl"
          style={{ background: `linear-gradient(90deg, transparent, ${color}, transparent)` }} />

        {/* Domain + Mastery */}
        <div className="flex items-center justify-between mb-6">
          <span className="text-[10px] font-black uppercase tracking-widest px-3 py-1.5 rounded-full border"
            style={{ color, borderColor: `${color}30`, backgroundColor: `${color}10` }}>
            {card.domain}
          </span>
          <MasteryRing score={card.mastery_score} size={64} />
        </div>

        {/* Question */}
        <div className="text-center mb-6">
          <p className="text-lg font-bold text-white leading-relaxed">{card.question || card.title}</p>
          {card.context && (
            <p className="text-sm text-gray-500 mt-2">{card.context}</p>
          )}
        </div>

        {/* Answer or Flip hint */}
        {revealed ? (
          <div className="mt-4 pt-4 border-t border-gray-800/60">
            <p className="text-base text-gray-200 text-center leading-relaxed whitespace-pre-wrap">{card.answer}</p>
          </div>
        ) : (
          <div className="flex items-center justify-center gap-2 text-gray-600 mt-4">
            <ArrowRight size={14} />
            <span className="text-sm">Tap to reveal answer</span>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Session Stats ────────────────────────────────────────────────────────────
function SessionStats({ stats }) {
  return (
    <div className="grid grid-cols-4 gap-3">
      {[
        { label: 'Reviewed', value: stats.reviewed, icon: CheckCircle2, color: 'text-emerald-400' },
        { label: 'Again',    value: stats.again,    icon: XCircle,      color: 'text-red-400' },
        { label: 'Streak',   value: stats.streak,   icon: Flame,        color: 'text-orange-400' },
        { label: 'XP',       value: `+${stats.xp}`, icon: Trophy,       color: 'text-yellow-400' },
      ].map(s => {
        const Icon = s.icon;
        return (
          <div key={s.label} className="p-3 rounded-xl border border-gray-800/50 bg-gray-900/40 text-center">
            <Icon size={16} className={`${s.color} mx-auto mb-1`} />
            <div className="text-lg font-black text-white">{s.value}</div>
            <div className="text-[10px] text-gray-600 uppercase tracking-wider">{s.label}</div>
          </div>
        );
      })}
    </div>
  );
}

// ─── Main Component ───────────────────────────────────────────────────────────
export default function StudyView() {
  const { API_BASE, user } = useWisdom();

  const [cards, setCards] = useState([]);
  const [currentIdx, setCurrentIdx] = useState(0);
  const [revealed, setRevealed] = useState(false);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [sessionDone, setSessionDone] = useState(false);
  const [stats, setStats] = useState({ reviewed: 0, again: 0, streak: 0, xp: 0 });

  const userId = user?.ldap || 'default';

  // ─── Load Due Cards ──────────────────────────────────────────────────────
  const loadCards = useCallback(async () => {
    setLoading(true);
    setCurrentIdx(0);
    setRevealed(false);
    setSessionDone(false);
    try {
      const res = await fetch(`${API_BASE}/api/v1/mastery/due?user_id=${userId}&limit=20`);
      if (res.ok) {
        const data = await res.json();
        setCards(data.cards || []);
      }
    } catch {}
    finally { setLoading(false); }
  }, [API_BASE, userId]);

  useEffect(() => {
    loadCards();
  }, [loadCards]);

  // ─── Submit Grade ────────────────────────────────────────────────────────
  const submitGrade = async (grade) => {
    const card = cards[currentIdx];
    if (!card || submitting) return;
    setSubmitting(true);

    // Optimistic XP update.
    const xp = grade === 1 ? 5 : grade === 2 ? 10 : grade === 3 ? 15 : 25;
    setStats(prev => ({
      reviewed: prev.reviewed + 1,
      again:    grade === 1 ? prev.again + 1 : prev.again,
      streak:   grade >= 3 ? prev.streak + 1 : 0,
      xp:       prev.xp + xp,
    }));

    try {
      await fetch(`${API_BASE}/api/v1/mastery/review`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          node_id: card.node_id,
          user_id: userId,
          grade,
          current_mastery_score: card.mastery_score,
        }),
      });
    } catch {}

    // Advance to next card.
    setSubmitting(false);
    if (currentIdx + 1 >= cards.length) {
      setSessionDone(true);
    } else {
      setCurrentIdx(i => i + 1);
      setRevealed(false);
    }
  };

  // ─── Keyboard Shortcuts ──────────────────────────────────────────────────
  useEffect(() => {
    const onKey = (e) => {
      if (!revealed) {
        if (e.code === 'Space') { e.preventDefault(); setRevealed(true); }
        return;
      }
      const grade = GRADES.find(g => g.key === e.key);
      if (grade) submitGrade(grade.value);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [revealed, currentIdx, submitGrade]);

  const currentCard = cards[currentIdx];
  const progress = cards.length > 0 ? ((currentIdx) / cards.length) * 100 : 0;

  // ─── Session Complete ────────────────────────────────────────────────────
  if (sessionDone) {
    return (
      <div className="h-full flex flex-col items-center justify-center bg-[#0d1117] p-8 gap-8">
        <div className="relative">
          <Trophy size={72} className="text-yellow-400" />
          <div className="absolute inset-0 blur-xl bg-yellow-400/20 rounded-full" />
        </div>
        <div className="text-center">
          <h2 className="text-3xl font-black text-white mb-2">Session Complete!</h2>
          <p className="text-gray-400">You reviewed {stats.reviewed} cards in this session.</p>
        </div>
        <SessionStats stats={stats} />
        <button onClick={loadCards}
          className="flex items-center gap-2 px-8 py-3 rounded-2xl bg-indigo-600 hover:bg-indigo-500 text-white font-bold transition-all shadow-lg shadow-indigo-500/20">
          <RotateCcw size={16} />
          New Session
        </button>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto bg-[#0d1117] text-gray-100">

      {/* Top Bar */}
      <div className="sticky top-0 z-10 px-6 pt-6 pb-4 bg-[#0d1117]/90 backdrop-blur border-b border-gray-800/40">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-3">
            <BookOpen className="text-indigo-400" size={20} />
            <h1 className="text-lg font-black text-white tracking-tight">Wisdom Study</h1>
          </div>
          <div className="flex items-center gap-3 text-sm text-gray-500">
            <span>{currentIdx} / {cards.length}</span>
            <button onClick={loadCards} className="p-1.5 rounded-lg hover:bg-gray-800/60 transition-colors">
              <RotateCcw size={14} />
            </button>
          </div>
        </div>

        {/* Progress bar */}
        <div className="h-1.5 bg-gray-800/80 rounded-full overflow-hidden">
          <div className="h-full bg-indigo-500 rounded-full transition-all duration-500"
            style={{ width: `${progress}%` }} />
        </div>
      </div>

      <div className="px-6 py-8 space-y-8 max-w-2xl mx-auto">

        {/* Stats Row */}
        <SessionStats stats={stats} />

        {/* Card */}
        {loading ? (
          <div className="flex items-center justify-center py-24">
            <Loader2 size={32} className="text-indigo-500 animate-spin" />
          </div>
        ) : cards.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-gray-600 rounded-3xl border border-gray-800/30">
            <Sparkles size={40} className="mb-4 text-indigo-400/40" />
            <p className="text-base font-bold text-gray-400">Nothing due today!</p>
            <p className="text-sm text-gray-600 mt-1">Come back later or explore the Knowledge Graph.</p>
          </div>
        ) : currentCard ? (
          <>
            <FlashCard card={currentCard} revealed={revealed} onReveal={() => setRevealed(true)} />

            {/* Grade Buttons */}
            {revealed && (
              <div className="space-y-4">
                <p className="text-center text-xs text-gray-600 uppercase tracking-widest">How well did you know this?</p>
                <div className="grid grid-cols-4 gap-3">
                  {GRADES.map(g => (
                    <button
                      key={g.value}
                      onClick={() => submitGrade(g.value)}
                      disabled={submitting}
                      className={`flex flex-col items-center gap-1.5 py-4 rounded-2xl border transition-all duration-200 ${g.color} disabled:opacity-50`}
                    >
                      <span className="font-black text-base">{g.label}</span>
                      <span className="text-[10px] opacity-70">{g.delta}</span>
                      <kbd className="text-[9px] opacity-50 border border-current/30 px-1.5 py-0.5 rounded">{g.key}</kbd>
                    </button>
                  ))}
                </div>
                <p className="text-center text-[10px] text-gray-700">Press 1–4 or Space to reveal</p>
              </div>
            )}
          </>
        ) : null}
      </div>
    </div>
  );
}
