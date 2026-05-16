import React, { useState, useEffect, useCallback } from 'react';
import { 
  Plus, Brain, Save, X, Sparkles, BookOpen, 
  Folder, MessageSquare, Tag, CheckCircle2 
} from 'lucide-react';
import { useWisdom } from '../context/WisdomContext';

export default function ExpertRegistry() {
  const { AGENT_BASE } = useWisdom();
  const [experts, setExperts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [isAdding, setIsAdding] = useState(false);
  const [success, setSuccess] = useState(null);
  
  const [formData, setFormData] = useState({
    id: '',
    description: '',
    keywords: '',
    system_instruction: '',
    anki_deck_prefix: '',
    obsidian_folder: ''
  });

  // Fetch existing domains
  const fetchDomains = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${AGENT_BASE}/domains`);
      const data = await res.json();
      const domainList = data.domains 
        ? Object.entries(data.domains).map(([id, cfg]) => ({ id, ...cfg }))
        : [];
      setExperts(domainList);
    } catch (err) {
      console.error('Failed to fetch domains:', err);
    } finally {
      setLoading(false);
    }
  }, [AGENT_BASE]);

  useEffect(() => {
    Promise.resolve().then(() => fetchDomains());
  }, [fetchDomains]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    
    const payload = {
      ...formData,
      keywords: formData.keywords.split(',').map(k => k.trim()).filter(k => k),
      id: formData.id.toUpperCase()
    };

    try {
      const res = await fetch(`${AGENT_BASE}/api/v1/experts`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      
      if (res.ok) {
        setSuccess(`Expert ${payload.id} registered successfully!`);
        setIsAdding(false);
        setFormData({
          id: '',
          description: '',
          keywords: '',
          system_instruction: '',
          anki_deck_prefix: '',
          obsidian_folder: ''
        });
        setTimeout(() => setSuccess(null), 3000);
        fetchDomains();
      } else {
        const err = await res.json();
        alert(`Error: ${err.detail || 'Failed to register expert'}`);
      }
    } catch (err) {
      alert(`Error: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const colors = [
    'from-indigo-500/20 to-purple-500/20',
    'from-emerald-500/20 to-teal-500/20',
    'from-cyan-500/20 to-blue-500/20',
    'from-orange-500/20 to-red-500/20',
    'from-pink-500/20 to-rose-500/20',
  ];

  return (
    <div className="h-full overflow-y-auto bg-[#0d1117] text-gray-100 p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-black text-white tracking-tight flex items-center gap-3">
            <Brain className="text-indigo-400" size={28} />
            Expert Registry
          </h1>
          <p className="text-sm text-gray-500 mt-0.5">Dynamic Domain Specialization & Cognitive Expansion</p>
        </div>
        <button
          onClick={() => setIsAdding(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white transition-all text-xs font-bold shadow-lg shadow-indigo-500/20"
        >
          <Plus size={16} />
          New Expert
        </button>
      </div>

      {success && (
        <div className="flex items-center gap-3 p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 animate-in fade-in slide-in-from-top-4 duration-300">
          <CheckCircle2 size={18} />
          <span className="text-sm font-medium">{success}</span>
        </div>
      )}

      {/* Registry Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {experts.map((expert, idx) => (
          <div key={expert.id} className="relative group p-5 rounded-2xl border border-gray-800/50 bg-gray-900/40 backdrop-blur hover:border-gray-700/60 transition-all overflow-hidden">
            <div className={`absolute inset-0 bg-gradient-to-br ${colors[idx % colors.length]} opacity-0 group-hover:opacity-100 transition-opacity duration-500`} />
            
            <div className="relative z-10">
              <div className="flex items-center justify-between mb-3">
                <div className="px-2 py-1 rounded-md bg-gray-800 text-[10px] font-black text-indigo-400 tracking-wider">
                  {expert.id}
                </div>
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]" />
                  <span className="text-[10px] text-gray-500 font-bold uppercase">Active</span>
                </div>
              </div>

              <h3 className="text-sm font-bold text-gray-100 mb-1">{expert.agent}</h3>
              <p className="text-[11px] text-gray-500 line-clamp-2 mb-4 leading-relaxed">
                {expert.description}
              </p>

              <div className="flex flex-wrap gap-1.5 mt-auto">
                {expert.keywords?.slice(0, 4).map(k => (
                  <span key={k} className="px-1.5 py-0.5 rounded bg-gray-800/80 text-[9px] text-gray-400 border border-gray-700/30">
                    #{k}
                  </span>
                ))}
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* New Expert Modal Overlay */}
      {isAdding && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="w-full max-w-2xl bg-[#161b22] border border-gray-800 rounded-3xl shadow-2xl overflow-hidden animate-in zoom-in-95 duration-300">
            <div className="p-6 border-b border-gray-800 flex items-center justify-between bg-gray-900/50">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-xl bg-indigo-500/10 text-indigo-400">
                  <Sparkles size={20} />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-white">Register Dynamic Expert</h2>
                  <p className="text-xs text-gray-500">Configure real-time cognitive routing for a new domain</p>
                </div>
              </div>
              <button 
                onClick={() => setIsAdding(false)}
                className="p-2 rounded-full hover:bg-gray-800 text-gray-500 transition-colors"
              >
                <X size={20} />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="p-6 space-y-5">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1.5">
                  <label className="text-[11px] font-bold text-gray-500 uppercase tracking-widest flex items-center gap-2">
                    <Tag size={12} /> Domain ID
                  </label>
                  <input
                    required
                    placeholder="e.g. COOKING"
                    className="w-full bg-gray-900/50 border border-gray-800 rounded-xl px-4 py-2.5 text-sm text-gray-200 focus:outline-none focus:border-indigo-500/50 transition-colors"
                    value={formData.id}
                    onChange={e => setFormData({ ...formData, id: e.target.value.toUpperCase() })}
                  />
                </div>
                <div className="space-y-1.5">
                  <label className="text-[11px] font-bold text-gray-500 uppercase tracking-widest flex items-center gap-2">
                    <MessageSquare size={12} /> Description
                  </label>
                  <input
                    required
                    placeholder="Briefly describe the domain..."
                    className="w-full bg-gray-900/50 border border-gray-800 rounded-xl px-4 py-2.5 text-sm text-gray-200 focus:outline-none focus:border-indigo-500/50 transition-colors"
                    value={formData.description}
                    onChange={e => setFormData({ ...formData, description: e.target.value })}
                  />
                </div>
              </div>

              <div className="space-y-1.5">
                <label className="text-[11px] font-bold text-gray-500 uppercase tracking-widest flex items-center gap-2">
                  <Sparkles size={12} /> Keywords (comma separated)
                </label>
                <input
                  placeholder="chef, recipe, kitchen, ingredients..."
                  className="w-full bg-gray-900/50 border border-gray-800 rounded-xl px-4 py-2.5 text-sm text-gray-200 focus:outline-none focus:border-indigo-500/50 transition-colors"
                  value={formData.keywords}
                  onChange={e => setFormData({ ...formData, keywords: e.target.value })}
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-[11px] font-bold text-gray-500 uppercase tracking-widest flex items-center gap-2">
                  <Brain size={12} /> System Instruction
                </label>
                <textarea
                  rows={4}
                  placeholder="Define the expert's personality and goals..."
                  className="w-full bg-gray-900/50 border border-gray-800 rounded-xl px-4 py-3 text-sm text-gray-200 focus:outline-none focus:border-indigo-500/50 transition-colors resize-none"
                  value={formData.system_instruction}
                  onChange={e => setFormData({ ...formData, system_instruction: e.target.value })}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1.5">
                  <label className="text-[11px] font-bold text-gray-500 uppercase tracking-widest flex items-center gap-2">
                    <BookOpen size={12} /> Anki Deck Prefix
                  </label>
                  <input
                    placeholder="Wisdom::Cooking (optional)"
                    className="w-full bg-gray-900/50 border border-gray-800 rounded-xl px-4 py-2.5 text-sm text-gray-200 focus:outline-none focus:border-indigo-500/50 transition-colors"
                    value={formData.anki_deck_prefix}
                    onChange={e => setFormData({ ...formData, anki_deck_prefix: e.target.value })}
                  />
                </div>
                <div className="space-y-1.5">
                  <label className="text-[11px] font-bold text-gray-500 uppercase tracking-widest flex items-center gap-2">
                    <Folder size={12} /> Obsidian Folder
                  </label>
                  <input
                    placeholder="Cooking/ (optional)"
                    className="w-full bg-gray-900/50 border border-gray-800 rounded-xl px-4 py-2.5 text-sm text-gray-200 focus:outline-none focus:border-indigo-500/50 transition-colors"
                    value={formData.obsidian_folder}
                    onChange={e => setFormData({ ...formData, obsidian_folder: e.target.value })}
                  />
                </div>
              </div>

              <div className="flex gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setIsAdding(false)}
                  className="flex-1 px-4 py-3 rounded-xl border border-gray-800 text-gray-400 hover:bg-gray-800 hover:text-white font-bold text-xs transition-all"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={loading}
                  className="flex-[2] px-4 py-3 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-bold text-xs transition-all shadow-lg shadow-indigo-500/20 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                >
                  {loading ? <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" /> : <Save size={16} />}
                  Save Expert
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
