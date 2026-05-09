import React, { useState, useEffect, useCallback } from 'react';
import { Activity, Zap, Clock, RefreshCw, BarChart3, TrendingUp } from 'lucide-react';
import { useWisdom } from '../context/WisdomContext';

const MetabolismView = () => {
  const { API_BASE, setLoading, setError } = useWisdom();
  const [report, setReport] = useState({ tsr: 0, metabolic_rate: 0, total_tokens: 0, signal_units: 0 });
  const [isRefreshing, setIsRefreshing] = useState(false);

  const fetchMetabolism = useCallback(async () => {
    setIsRefreshing(true);
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE}/metabolism`);
      if (!response.ok) throw new Error("Failed to audit metabolism");
      const result = await response.json();
      setReport(result);
    } catch (error) {
      console.error("Failed to fetch metabolism data:", error);
      setError(error.message);
    } finally {
      setIsRefreshing(false);
      setLoading(false);
    }
  }, [API_BASE, setLoading, setError]);

  useEffect(() => {
    fetchMetabolism();
  }, [fetchMetabolism]);


  return (
    <div className="p-10 space-y-10 bg-[#0d1117] min-h-full overflow-y-auto custom-scrollbar text-gray-100">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-black text-white tracking-tighter flex items-center gap-4">
            <div className="p-2 bg-indigo-500/10 rounded-xl border border-indigo-500/20">
              <Activity className="text-indigo-400" size={32} />
            </div>
            Metabolic Audit
          </h1>
          <p className="text-gray-500 text-sm font-medium mt-2 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-indigo-400 animate-pulse" />
            Quantifying cognitive resource efficiency (TSR)
          </p>
        </div>
        <button 
          onClick={fetchMetabolism}
          className="flex items-center gap-2.5 px-6 py-3 bg-gray-900/50 border border-gray-800 rounded-2xl text-xs font-black text-gray-400 hover:text-white hover:border-indigo-500/50 hover:bg-indigo-500/5 transition-all active:scale-95 shadow-xl group"
        >
          <RefreshCw size={16} className={`${isRefreshing ? 'animate-spin' : 'group-hover:rotate-180 transition-transform duration-500'}`} />
          {isRefreshing ? 'SYNCHRONIZING...' : 'FORCE SYNC'}
        </button>
      </div>

      {/* Primary Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <StatCard 
          icon={<TrendingUp className="text-green-400" size={20} />} 
          label="Metabolic Efficiency" 
          value={(report.tsr * 100).toFixed(1)} 
          suffix="%"
          sublabel="Token-to-Signal Ratio"
        />
        <StatCard 
          icon={<Zap className="text-indigo-400" size={20} />} 
          label="Total Consumption" 
          value={report.total_tokens.toLocaleString()} 
          suffix="Tokens"
          sublabel="Aggregate Context Usage"
        />
        <StatCard 
          icon={<Activity className="text-purple-400" size={20} />} 
          label="Metabolic Rate" 
          value={report.metabolic_rate.toFixed(2)} 
          suffix="T/s"
          sublabel="Tokens per Second"
        />
        <StatCard 
          icon={<BarChart3 className="text-amber-400" size={20} />} 
          label="Signal Harvest" 
          value={report.signal_units} 
          suffix="Units"
          sublabel="High-Value Outcomes"
        />
      </div>

      {/* Detail Section */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <div className="bg-black/20 border border-gray-800 rounded-3xl p-8 space-y-6">
          <h2 className="text-lg font-bold text-white flex items-center gap-3">
            <TrendingUp size={20} className="text-indigo-400" />
            Efficiency Analysis
          </h2>
          <div className="space-y-4">
            <p className="text-sm text-gray-400 leading-relaxed">
              The Token-to-Signal Ratio (TSR) measures how effectively Wisdom converts LLM context into verified SRE knowledge. 
              A higher percentage indicates more efficient grounding and less &quot;hallucinatory noise&quot;.
            </p>
            <div className="h-2 w-full bg-gray-800 rounded-full overflow-hidden">
               <div 
                className="h-full bg-indigo-500 shadow-[0_0_20px_rgba(99,102,241,0.5)] transition-all duration-1000" 
                style={{ width: `${Math.min(report.tsr * 100, 100)}%` }}
               />
            </div>
          </div>
        </div>

        <div className="bg-black/20 border border-gray-800 rounded-3xl p-8 space-y-6">
          <h2 className="text-lg font-bold text-white flex items-center gap-3">
            <Clock size={20} className="text-indigo-400" />
            System Health
          </h2>
          <div className="grid grid-cols-2 gap-4">
             <div className="p-4 bg-gray-900/50 border border-gray-800 rounded-2xl">
                <span className="text-[10px] font-black text-gray-500 uppercase tracking-widest block mb-1">State</span>
                <span className="text-sm font-bold text-green-400">NOMINAL</span>
             </div>
             <div className="p-4 bg-gray-900/50 border border-gray-800 rounded-2xl">
                <span className="text-[10px] font-black text-gray-500 uppercase tracking-widest block mb-1">Uptime</span>
                <span className="text-sm font-bold text-white">99.9%</span>
             </div>
          </div>
        </div>
      </div>
    </div>
  );
};

const StatCard = ({ icon, label, value, suffix, sublabel }) => (
  <div className="bg-black/20 border border-gray-800 rounded-3xl p-6 hover:border-indigo-500/30 transition-all group shadow-xl">
    <div className="flex items-center gap-3 mb-4">
      <div className="p-2 bg-gray-900 rounded-xl border border-gray-800 group-hover:scale-110 transition-transform">
        {icon}
      </div>
      <span className="text-[10px] font-black text-gray-500 uppercase tracking-widest">{label}</span>
    </div>
    <div className="flex items-baseline gap-2">
      <span className="text-3xl font-black text-white tabular-nums">{value}</span>
      <span className="text-xs font-bold text-gray-600 uppercase">{suffix}</span>
    </div>
    <p className="text-[10px] font-bold text-gray-600 mt-2 uppercase tracking-tighter">{sublabel}</p>
  </div>
);

export default MetabolismView;
