import React, { useState, useEffect, useCallback } from 'react';
import {
  Globe, RefreshCw, CheckCircle2,
  ExternalLink, Loader2, Play, Rss, Tag, Link2
} from 'lucide-react';
import { useWisdom } from '../context/WisdomContext';

// ─── Job Status Badge ─────────────────────────────────────────────────────────
function JobBadge({ status }) {
  const map = {
    RUNNING:    { color: 'text-cyan-400 bg-cyan-500/10 border-cyan-500/20', dot: 'bg-cyan-400 animate-pulse' },
    COMPLETED:  { color: 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20', dot: 'bg-emerald-400' },
    FAILED:     { color: 'text-red-400 bg-red-500/10 border-red-500/20', dot: 'bg-red-400' },
    QUEUED:     { color: 'text-yellow-400 bg-yellow-500/10 border-yellow-500/20', dot: 'bg-yellow-400' },
    SCRAPING:   { color: 'text-indigo-400 bg-indigo-500/10 border-indigo-500/20', dot: 'bg-indigo-400 animate-pulse' },
  };
  const cfg = map[status] || map.QUEUED;
  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-bold border ${cfg.color}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${cfg.dot}`} />
      {status}
    </span>
  );
}

// ─── Progress Bar ─────────────────────────────────────────────────────────────
function ProgressBar({ progress, color = '#6366f1' }) {
  return (
    <div className="h-1.5 bg-gray-800/80 rounded-full overflow-hidden">
      <div
        className="h-full rounded-full transition-all duration-500"
        style={{ width: `${Math.min(100, progress)}%`, background: color }}
      />
    </div>
  );
}

// ─── Ingested Article Card ────────────────────────────────────────────────────
function ArticleCard({ article }) {
  return (
    <div className="p-3 rounded-xl border border-gray-800/50 bg-gray-900/30 hover:border-gray-700/60 transition-all group">
      <div className="flex items-start justify-between gap-2 mb-2">
        <p className="text-xs font-semibold text-gray-200 line-clamp-2 leading-snug">{article.title}</p>
        {article.url && (
          <a href={article.url} target="_blank" rel="noopener noreferrer"
            className="flex-shrink-0 p-1 rounded-lg hover:bg-gray-700/60 text-gray-600 hover:text-indigo-400 transition-colors"
          >
            <ExternalLink size={12} />
          </a>
        )}
      </div>
      <div className="flex items-center gap-2 flex-wrap">
        {article.tags?.slice(0, 3).map(tag => (
          <span key={tag} className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-indigo-500/10 border border-indigo-500/20 text-[9px] font-bold text-indigo-400">
            <Tag size={8} />
            {tag}
          </span>
        ))}
      </div>
      <div className="flex items-center justify-between mt-2">
        <span className="text-[10px] text-gray-600">{article.domain}</span>
        <span className="text-[10px] text-gray-600">{article.ingested_at}</span>
      </div>
    </div>
  );
}

// ─── Research Job Card ────────────────────────────────────────────────────────
function JobCard({ job }) {
  const domainColors = {
    CHESS: '#06b6d4', FINANCE: '#10b981', LANGUAGE: '#f59e0b', TECH: '#6366f1', GENERAL: '#8b5cf6',
  };
  const color = domainColors[job.domain] || '#8b5cf6';

  return (
    <div className="p-4 rounded-2xl border border-gray-800/50 bg-gray-900/40 backdrop-blur hover:border-gray-700/60 transition-all">
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <div className="w-2 h-2 rounded-full flex-shrink-0" style={{ backgroundColor: color }} />
            <span className="text-[10px] font-bold uppercase tracking-wider" style={{ color }}>{job.domain}</span>
          </div>
          <p className="text-sm font-bold text-gray-200 truncate">{job.topic}</p>
        </div>
        <JobBadge status={job.status} />
      </div>

      <ProgressBar progress={job.progress || 0} color={color} />

      <div className="flex items-center justify-between mt-2">
        <div className="flex items-center gap-3 text-[10px] text-gray-600">
          <span className="flex items-center gap-1"><Link2 size={9} />{job.urls_scraped || 0} urls</span>
          <span className="flex items-center gap-1"><CheckCircle2 size={9} />{job.nodes_created || 0} nodes</span>
        </div>
        <span className="text-[10px] text-gray-600">
          {job.status === 'RUNNING' ? `${job.progress || 0}%` : job.completed_at || '—'}
        </span>
      </div>
    </div>
  );
}

// ─── Main Component ───────────────────────────────────────────────────────────
export default function ResearcherView() {
  const { API_BASE, lastEvent } = useWisdom();

  const [jobs, setJobs] = useState([]);
  const [recentArticles, setRecentArticles] = useState([]);
  const [loading, setLoading] = useState(true);
  const [newTopic, setNewTopic] = useState('');
  const [newDomain, setNewDomain] = useState('TECH');
  const [submitting, setSubmitting] = useState(false);

  // ─── Fetch State ──────────────────────────────────────────────────────────
  const fetchJobs = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/research/jobs`);
      if (res.ok) setJobs(await res.json());
    } catch (err) {
      console.error('Failed to fetch jobs:', err);
    }
  }, [API_BASE]);

  const fetchArticles = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/research/recent?limit=12`);
      if (res.ok) {
        const data = await res.json();
        setRecentArticles(data.articles || []);
      }
    } catch (err) {
      console.error('Failed to fetch articles:', err);
    }
    finally { setLoading(false); }
  }, [API_BASE]);

  useEffect(() => {
    Promise.resolve().then(() => {
      fetchJobs();
      fetchArticles();
    });
    const interval = setInterval(fetchJobs, 10_000);
    return () => clearInterval(interval);
  }, [fetchJobs, fetchArticles]);

  // ─── Live Job Updates via WebSocket ──────────────────────────────────────
  useEffect(() => {
    if (!lastEvent) return;
    if (lastEvent.type === 'wisdom.researcher.scrape_progress') {
      Promise.resolve().then(() => {
        setJobs(prev => prev.map(j =>
          j.id === lastEvent.job_id ? { ...j, ...lastEvent } : j
        ));
      });
    }
    if (lastEvent.type === 'wisdom.knowledge.ingested') {
      Promise.resolve().then(() => {
        setRecentArticles(prev => [lastEvent, ...prev].slice(0, 12));
      });
    }
  }, [lastEvent]);

  // ─── Trigger Research Job ────────────────────────────────────────────────
  const submitJob = async () => {
    if (!newTopic.trim()) return;
    setSubmitting(true);
    try {
      await fetch(`${API_BASE}/api/v1/research/investigate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ topic: newTopic, domain: newDomain, depth: 2 }),
      });
      setNewTopic('');
      setTimeout(fetchJobs, 1000);
    } catch (err) {
      console.error('Failed to submit job:', err);
    }
    finally { setSubmitting(false); }
  };

  const activeJobs = jobs.filter(j => ['RUNNING', 'SCRAPING', 'QUEUED'].includes(j.status));
  const completedJobs = jobs.filter(j => ['COMPLETED', 'FAILED'].includes(j.status));

  return (
    <div className="h-full overflow-y-auto bg-[#0d1117] text-gray-100 p-6 space-y-6">

      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-black text-white tracking-tight flex items-center gap-3">
            <Rss className="text-green-400" size={22} />
            Researcher Monitor
          </h1>
          <p className="text-sm text-gray-500 mt-0.5">Autonomous content ingestion pipeline</p>
        </div>
        <button onClick={() => { fetchJobs(); fetchArticles(); }}
          className="flex items-center gap-2 px-4 py-2 rounded-xl bg-gray-800/60 border border-gray-700/50 text-gray-400 hover:text-white hover:border-emerald-500/40 transition-all text-xs font-bold">
          <RefreshCw size={13} />
          Refresh
        </button>
      </div>

      {/* Trigger Research */}
      <div className="p-4 rounded-2xl border border-gray-800/50 bg-gray-900/40">
        <div className="text-xs font-black text-gray-400 uppercase tracking-widest mb-3">Trigger Research Job</div>
        <div className="flex gap-3">
          <select
            value={newDomain}
            onChange={e => setNewDomain(e.target.value)}
            className="bg-gray-800/80 border border-gray-700/50 rounded-xl text-sm text-gray-300 px-3 py-2.5 focus:outline-none focus:border-green-500/40"
          >
            {['CHESS', 'FINANCE', 'LANGUAGE', 'TECH', 'GENERAL'].map(d => (
              <option key={d} value={d}>{d}</option>
            ))}
          </select>
          <input
            value={newTopic}
            onChange={e => setNewTopic(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && submitJob()}
            placeholder="Enter topic to research, e.g. 'Caro-Kann Defense history'"
            className="flex-1 bg-gray-800/80 border border-gray-700/50 rounded-xl text-sm text-gray-300 px-4 py-2.5 focus:outline-none focus:border-green-500/40 placeholder:text-gray-600"
          />
          <button
            onClick={submitJob}
            disabled={submitting || !newTopic.trim()}
            className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-green-500/20 border border-green-500/30 text-green-400 hover:bg-green-500/30 transition-all font-bold text-sm disabled:opacity-50"
          >
            {submitting ? <Loader2 size={14} className="animate-spin" /> : <Play size={14} />}
            Research
          </button>
        </div>
      </div>

      {/* Active Jobs */}
      {activeJobs.length > 0 && (
        <div>
          <div className="flex items-center gap-2 mb-3">
            <h2 className="text-sm font-black text-gray-300 uppercase tracking-widest">Active Jobs</h2>
            <div className="w-2 h-2 rounded-full bg-cyan-400 animate-pulse" />
          </div>
          <div className="grid grid-cols-2 gap-3">
            {activeJobs.map(job => <JobCard key={job.id} job={job} />)}
          </div>
        </div>
      )}

      {/* Recent Ingestions */}
      <div>
        <h2 className="text-sm font-black text-gray-300 uppercase tracking-widest mb-3">Recently Ingested</h2>
        {loading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 size={24} className="text-indigo-500 animate-spin" />
          </div>
        ) : recentArticles.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-gray-600 rounded-2xl border border-gray-800/40 bg-gray-900/20">
            <Globe size={32} className="mb-3 opacity-30" />
            <p className="text-sm">No articles ingested yet</p>
            <p className="text-xs mt-1 text-gray-700">Trigger a research job above to start ingestion</p>
          </div>
        ) : (
          <div className="grid grid-cols-3 gap-3">
            {recentArticles.map((article, i) => <ArticleCard key={i} article={article} />)}
          </div>
        )}
      </div>

      {/* Completed Jobs */}
      {completedJobs.length > 0 && (
        <div>
          <h2 className="text-sm font-black text-gray-300 uppercase tracking-widest mb-3">History</h2>
          <div className="grid grid-cols-2 gap-3">
            {completedJobs.slice(0, 6).map(job => <JobCard key={job.id} job={job} />)}
          </div>
        </div>
      )}
    </div>
  );
}
