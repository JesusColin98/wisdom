import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Activity, CheckCircle2, XCircle, Clock, AlertTriangle,
  RefreshCw, Zap, Server, Brain, Database, Globe,
  ArrowRight, Wifi, WifiOff
} from 'lucide-react';
import { useWisdom } from '../context/WisdomContext';

// ─── Service Definitions ──────────────────────────────────────────────────────
const SERVICES = [
  { id: 'cortex',       label: 'Cortex',       icon: Database, port: 50051, color: '#6366f1', type: 'go' },
  { id: 'thalamus',     label: 'Thalamus',      icon: Brain,    port: 50052, color: '#8b5cf6', type: 'go' },
  { id: 'mastery',      label: 'Mastery',       icon: Activity, port: 50053, color: '#06b6d4', type: 'go' },
  { id: 'researcher',   label: 'Researcher',    icon: Globe,    port: 50054, color: '#10b981', type: 'go' },
  { id: 'curriculum',   label: 'Curriculum',    icon: Zap,      port: 50055, color: '#f59e0b', type: 'go' },
  { id: 'integrations', label: 'Integrations',  icon: ArrowRight, port: 50056, color: '#ec4899', type: 'go' },
  { id: 'entity',       label: 'Entity',        icon: Server,   port: 50057, color: '#84cc16', type: 'go' },
  { id: 'adk-router',   label: 'ADK Router',    icon: Brain,    port: 8081,  color: '#f97316', type: 'python' },
];

const MCP_SERVERS = [
  { id: 'obsidian-mcp', label: 'Obsidian MCP', port: 3333, color: '#a78bfa' },
  { id: 'anki-mcp',     label: 'Anki MCP',     port: 3334, color: '#34d399' },
];

// ─── Status Badge ─────────────────────────────────────────────────────────────
function StatusBadge({ status }) {
  const map = {
    online:   { icon: CheckCircle2, color: 'text-emerald-400', bg: 'bg-emerald-500/10 border-emerald-500/20', label: 'Online' },
    offline:  { icon: XCircle,      color: 'text-red-400',     bg: 'bg-red-500/10 border-red-500/20',         label: 'Offline' },
    checking: { icon: RefreshCw,    color: 'text-yellow-400',  bg: 'bg-yellow-500/10 border-yellow-500/20',  label: 'Checking' },
    unknown:  { icon: AlertTriangle, color: 'text-gray-400',   bg: 'bg-gray-700/30 border-gray-700/40',      label: 'Unknown' },
  };
  const cfg = map[status] || map.unknown;
  const Icon = cfg.icon;
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-bold border ${cfg.bg} ${cfg.color}`}>
      <Icon size={10} className={status === 'checking' ? 'animate-spin' : ''} />
      {cfg.label}
    </span>
  );
}

// ─── Routing Log Entry ────────────────────────────────────────────────────────
function RoutingLogEntry({ entry }) {
  const domainColors = {
    CHESS: '#06b6d4', FINANCE: '#10b981', LANGUAGE: '#f59e0b',
    TECH: '#6366f1', GENERAL: '#8b5cf6',
  };
  const color = domainColors[entry.domain] || '#8b5cf6';

  return (
    <div className="flex items-center gap-3 py-2.5 border-b border-gray-800/40 group hover:bg-gray-800/20 px-3 rounded-lg transition-colors">
      <div className="w-1.5 h-1.5 rounded-full flex-shrink-0" style={{ backgroundColor: color, boxShadow: `0 0 6px ${color}` }} />
      <div className="flex-1 min-w-0">
        <div className="text-[11px] text-gray-300 truncate">{entry.input?.slice(0, 60)}...</div>
        <div className="flex items-center gap-2 mt-0.5">
          <span className="text-[10px] font-bold" style={{ color }}>{entry.domain}</span>
          <span className="text-[10px] text-gray-600">→</span>
          <span className="text-[10px] text-gray-500">{entry.agent}</span>
        </div>
      </div>
      <div className="flex flex-col items-end gap-0.5">
        <span className="text-[10px] text-gray-500">{entry.elapsed_ms}ms</span>
        <span className="text-[10px] text-gray-600">{(entry.confidence * 100).toFixed(0)}%</span>
      </div>
    </div>
  );
}

// ─── Service Card ─────────────────────────────────────────────────────────────
function ServiceCard({ service, status, latency }) {
  const Icon = service.icon;
  return (
    <div className="relative p-4 rounded-2xl border border-gray-800/50 bg-gray-900/40 backdrop-blur hover:border-gray-700/60 transition-all group overflow-hidden">
      <div
        className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500"
        style={{ background: `radial-gradient(ellipse at 50% 0%, ${service.color}08, transparent 70%)` }}
      />
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2.5">
          <div className="p-2 rounded-xl" style={{ backgroundColor: `${service.color}15` }}>
            <Icon size={16} style={{ color: service.color }} />
          </div>
          <div>
            <div className="text-xs font-bold text-gray-200">{service.label}</div>
            <div className="text-[10px] text-gray-600">:{service.port}</div>
          </div>
        </div>
        <StatusBadge status={status} />
      </div>
      <div className="flex items-center justify-between">
        <span className="text-[10px] text-gray-600 uppercase tracking-wider">{service.type}</span>
        {latency && status === 'online' && (
          <span className="text-[10px] text-emerald-400">{latency}ms</span>
        )}
      </div>
      {status === 'online' && (
        <div className="absolute bottom-0 left-0 right-0 h-0.5 rounded-b-2xl"
          style={{ background: `linear-gradient(90deg, transparent, ${service.color}, transparent)` }} />
      )}
    </div>
  );
}

// ─── Main Component ───────────────────────────────────────────────────────────
export default function MissionControlView() {
  const { API_BASE, AGENT_BASE, lastEvent } = useWisdom();

  const [serviceStatus, setServiceStatus] = useState({});
  const [latencies, setLatencies] = useState({});
  const [routingLog, setRoutingLog] = useState([]);
  const [pubsubConnected, setPubsubConnected] = useState(false);
  const [systemMetrics, setSystemMetrics] = useState({ totalRouted: 0, avgLatency: 0, uptime: 0 });
  const esRef = useRef(null);

  // ─── Health Polling ─────────────────────────────────────────────────────────
  const checkService = useCallback(async (svc) => {
    const base = svc.type === 'python' ? AGENT_BASE : API_BASE;
    const start = Date.now();
    try {
      const res = await fetch(`${base}/health`, { signal: AbortSignal.timeout(3000) });
      const latency = Date.now() - start;
      setServiceStatus(p => ({ ...p, [svc.id]: res.ok ? 'online' : 'offline' }));
      if (res.ok) setLatencies(p => ({ ...p, [svc.id]: latency }));
    } catch {
      setServiceStatus(p => ({ ...p, [svc.id]: 'offline' }));
    }
  }, [API_BASE, AGENT_BASE]);

  const pollAll = useCallback(() => {
    SERVICES.forEach(svc => {
      setServiceStatus(p => ({ ...p, [svc.id]: p[svc.id] || 'checking' }));
      checkService(svc);
    });
  }, [checkService]);

  useEffect(() => {
    pollAll();
    const interval = setInterval(pollAll, 30_000);
    return () => clearInterval(interval);
  }, [pollAll]);

  // ─── Routing Log via SSE ────────────────────────────────────────────────────
  useEffect(() => {
    const es = new EventSource(`${AGENT_BASE}/events/routing`);
    esRef.current = es;

    es.onopen = () => setPubsubConnected(true);
    es.onerror = () => setPubsubConnected(false);

    es.addEventListener('routing_decision', (e) => {
      try {
        const entry = JSON.parse(e.data);
        setRoutingLog(prev => [entry, ...prev].slice(0, 50));
        setSystemMetrics(prev => ({
          totalRouted: prev.totalRouted + 1,
          avgLatency: Math.round((prev.avgLatency + (entry.elapsed_ms || 0)) / 2),
          uptime: prev.uptime,
        }));
      } catch {}
    });

    return () => es.close();
  }, [AGENT_BASE]);

  // ─── WebSocket events from Thalamus ────────────────────────────────────────
  useEffect(() => {
    if (!lastEvent) return;
    if (lastEvent.type === 'wisdom.router.decision_logged') {
      setRoutingLog(prev => [lastEvent, ...prev].slice(0, 50));
    }
  }, [lastEvent]);

  // ─── Derived Stats ──────────────────────────────────────────────────────────
  const onlineCount = Object.values(serviceStatus).filter(s => s === 'online').length;
  const offlineCount = Object.values(serviceStatus).filter(s => s === 'offline').length;

  return (
    <div className="h-full overflow-y-auto bg-[#0d1117] text-gray-100 p-6 space-y-6">

      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-black text-white tracking-tight">Mission Control</h1>
          <p className="text-sm text-gray-500 mt-0.5">Cognitive Runtime Health & Routing Observatory</p>
        </div>
        <button
          onClick={pollAll}
          className="flex items-center gap-2 px-4 py-2 rounded-xl bg-gray-800/60 border border-gray-700/50 text-gray-400 hover:text-white hover:border-indigo-500/40 transition-all text-xs font-bold"
        >
          <RefreshCw size={13} />
          Refresh
        </button>
      </div>

      {/* System KPIs */}
      <div className="grid grid-cols-4 gap-4">
        {[
          { label: 'Services Online', value: `${onlineCount} / ${SERVICES.length}`, color: 'emerald', icon: CheckCircle2 },
          { label: 'Services Offline', value: offlineCount, color: 'red', icon: XCircle },
          { label: 'Total Routed', value: systemMetrics.totalRouted, color: 'indigo', icon: ArrowRight },
          { label: 'Avg Latency', value: `${systemMetrics.avgLatency}ms`, color: 'cyan', icon: Clock },
        ].map(kpi => {
          const Icon = kpi.icon;
          return (
            <div key={kpi.label} className="p-4 rounded-2xl border border-gray-800/50 bg-gray-900/40 backdrop-blur">
              <div className={`text-${kpi.color}-400 mb-2`}><Icon size={18} /></div>
              <div className="text-2xl font-black text-white">{kpi.value}</div>
              <div className="text-[11px] text-gray-500 mt-1">{kpi.label}</div>
            </div>
          );
        })}
      </div>

      {/* Service Grid */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-black text-gray-300 uppercase tracking-widest">Microservices</h2>
          <div className="flex items-center gap-1.5">
            {pubsubConnected
              ? <><Wifi size={12} className="text-emerald-400" /><span className="text-[10px] text-emerald-400">Live</span></>
              : <><WifiOff size={12} className="text-red-400" /><span className="text-[10px] text-red-400">Offline</span></>
            }
          </div>
        </div>
        <div className="grid grid-cols-4 gap-3">
          {SERVICES.map(svc => (
            <ServiceCard
              key={svc.id}
              service={svc}
              status={serviceStatus[svc.id] || 'checking'}
              latency={latencies[svc.id]}
            />
          ))}
        </div>
      </div>

      {/* MCP Servers */}
      <div>
        <h2 className="text-sm font-black text-gray-300 uppercase tracking-widest mb-3">Local MCP Servers</h2>
        <div className="grid grid-cols-2 gap-3">
          {MCP_SERVERS.map(mcp => (
            <div key={mcp.id} className="flex items-center justify-between p-4 rounded-2xl border border-gray-800/50 bg-gray-900/40">
              <div className="flex items-center gap-3">
                <div className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: mcp.color }} />
                <div>
                  <div className="text-xs font-bold text-gray-200">{mcp.label}</div>
                  <div className="text-[10px] text-gray-600">localhost:{mcp.port}</div>
                </div>
              </div>
              <StatusBadge status={serviceStatus[mcp.id] || 'unknown'} />
            </div>
          ))}
        </div>
      </div>

      {/* Routing Log */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-black text-gray-300 uppercase tracking-widest">Routing Log</h2>
          <span className="text-[10px] text-gray-600">{routingLog.length} events</span>
        </div>
        <div className="rounded-2xl border border-gray-800/50 bg-gray-900/20 backdrop-blur overflow-hidden">
          {routingLog.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-gray-600">
              <Activity size={32} className="mb-3 opacity-30" />
              <p className="text-sm">Waiting for routing events...</p>
              <p className="text-xs mt-1 text-gray-700">Voice input → ADK Router → Domain Expert</p>
            </div>
          ) : (
            <div className="divide-y divide-gray-800/20 max-h-80 overflow-y-auto">
              {routingLog.map((entry, i) => (
                <RoutingLogEntry key={i} entry={entry} />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
iv>
          )}
        </div>
      </div>
    </div>
  );
}
