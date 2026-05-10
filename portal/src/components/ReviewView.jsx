import React, { useState, useEffect, useCallback } from 'react';
import { Sparkles, CheckCircle, Brain } from 'lucide-react';
import { useWisdom } from '../context/WisdomContext';

const ReviewView = () => {
  const { API_BASE, setLoading, setError, activeNamespace } = useWisdom();
  const [dueNodes, setDueNodes] = useState([]);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [showContent, setShowContent] = useState(false);

  const fetchDueNodes = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/cortex/due?namespace=${activeNamespace}`);
      if (res.ok) {
        const data = await res.json();
        setDueNodes(data || []);
      }
    } catch (e) { setError(e.message); }
    finally { setLoading(false); }
  }, [API_BASE, activeNamespace, setLoading, setError]);

  useEffect(() => {
    const timer = setTimeout(() => {
      fetchDueNodes();
    }, 0);
    return () => clearTimeout(timer);
  }, [fetchDueNodes]);

  const handleReview = async (grade) => {
    const node = dueNodes[currentIndex];
    try {
        const res = await fetch(`${API_BASE}/cortex/review`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ node_id: node.id, grade })
        });
        if (res.ok) {
            if (currentIndex < dueNodes.length - 1) {
                setCurrentIndex(currentIndex + 1);
                setShowContent(false);
            } else {
                setDueNodes([]);
                window.alert("Review session complete!");
            }
        }
    } catch (e) { window.alert(e.message); }
  };

  if (dueNodes.length === 0) {
    return (
      <div className="h-full flex items-center justify-center bg-[#0d1117]">
        <div className="text-center space-y-4">
            <div className="p-6 bg-indigo-500/5 rounded-full inline-block border border-indigo-500/10">
                <CheckCircle className="text-indigo-400" size={48} />
            </div>
            <h2 className="text-2xl font-bold text-white">All caught up!</h2>
            <p className="text-gray-500 text-sm">No knowledge nodes are due for review in this namespace.</p>
            <button onClick={fetchDueNodes} className="px-6 py-2 bg-indigo-600 text-white rounded-xl font-bold text-xs uppercase tracking-widest mt-4">Refresh</button>
        </div>
      </div>
    );
  }

  const currentNode = dueNodes[currentIndex];

  return (
    <div className="h-full flex flex-col items-center justify-center bg-[#0d1117] p-8">
        <div className="max-w-2xl w-full space-y-8">
            <div className="flex justify-between items-center">
                <div className="flex items-center gap-3">
                    <Sparkles className="text-indigo-400" size={24} />
                    <span className="text-xs font-black text-gray-500 uppercase tracking-widest">Neural Reinforcement ({currentIndex + 1}/{dueNodes.length})</span>
                </div>
                <div className="px-3 py-1 bg-indigo-500/10 border border-indigo-500/20 rounded-full text-[10px] font-bold text-indigo-300">
                    Namespace: {activeNamespace}
                </div>
            </div>

            <div className={`w-full aspect-[4/3] bg-gray-900/50 border-2 ${showContent ? 'border-indigo-500/40' : 'border-gray-800'} rounded-[3rem] p-12 flex flex-col items-center justify-center text-center shadow-2xl transition-all duration-500 relative overflow-hidden group`}>
                {!showContent ? (
                    <div className="space-y-6 animate-in fade-in zoom-in-95 duration-500">
                        <h1 className="text-4xl font-black text-white tracking-tighter">{currentNode.id}</h1>
                        <p className="text-gray-400 font-medium italic">Can you recall the associated knowledge?</p>
                        <button 
                            onClick={() => setShowContent(true)}
                            className="px-8 py-4 bg-indigo-600 hover:bg-indigo-500 text-white rounded-2xl font-black text-xs uppercase tracking-[0.2em] shadow-xl shadow-indigo-500/20 transition-all active:scale-95"
                        >
                            Reveal Fact
                        </button>
                    </div>
                ) : (
                    <div className="w-full h-full flex flex-col items-center justify-center space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700">
                        <div className="p-8 bg-black/40 rounded-3xl border border-white/5 w-full max-h-full overflow-y-auto custom-scrollbar">
                            <p className="text-xl leading-relaxed text-gray-200 font-serif">{currentNode.content}</p>
                        </div>
                        <div className="flex gap-3">
                            {[
                                { grade: 0, label: 'Blackout', color: 'bg-red-500/20 text-red-400 border-red-500/30' },
                                { grade: 3, label: 'Recall', color: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30' },
                                { grade: 5, label: 'Perfect', color: 'bg-green-500/20 text-green-400 border-green-500/30' }
                            ].map(btn => (
                                <button 
                                    key={btn.grade}
                                    onClick={() => handleReview(btn.grade)}
                                    className={`px-6 py-3 ${btn.color} border rounded-xl font-black text-[10px] uppercase tracking-widest hover:scale-105 transition-all active:scale-95`}
                                >
                                    {btn.label}
                                </button>
                            ))}
                        </div>
                    </div>
                )}
                
                {/* Decorative element */}
                <div className="absolute top-0 right-0 p-8 opacity-10 group-hover:opacity-20 transition-opacity">
                    <Brain size={120} />
                </div>
            </div>
        </div>
    </div>
  );
};

export default ReviewView;
