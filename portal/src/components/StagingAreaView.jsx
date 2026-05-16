import React, { useState, useEffect, useCallback } from 'react';
import {
  CloudOff, RefreshCw, RotateCcw, CheckCircle2, XCircle,
  FileText, CreditCard, Loader2, AlertTriangle, Clock, Wifi
} from 'lucide-react';
import { useWisdom } from '../context/WisdomContext';

// ─── Type Badge ───────────────────────────────────────────────────────────────
function TypeBadge({ type, app }) {
  const config = {
    NOTE: { icon: FileText, color: 'text-violet-400 bg-violet-500/10 border-violet-500/20' },
    CARD: { icon: CreditCard, color: 'text-cyan-400 bg-cyan-500/10 border-cyan-500/20' },
  };
  const appColor = {
    OBSIDIAN: 'text-purple-400',
    ANKI: 'text-red-400',
  };
  const cfg = config[type] || config.NOTE;
  const Icon = cfg.icon;
  return (
    <div className="flex items-center gap-1.5">
      <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-bold border ${cfg.color}`}>
        <Icon size={9} />
        {type}
      </span>
      <span className={`text-[10px] font-bold ${appColor[app] || 'text-gray-500'}`}>→ {app}</span>
    </div>
  );
}

// ─── Pending Item Card ────────────────────────────────────────────────────────
function PendingItemCard({ item, onRetry, onDismiss, retrying }) {
  const payload = (() => {
    try { return JSON.parse(item.payload_json || '{}'); } catch { return {}; }
  })();

  return (
    <div className="p-4 rounded-2xl border border-yellow-500/20 bg-yellow-500/5 backdrop-blur hover:border-yellow-500/30 transition-all">
      <div className="flex items-start justify-between mb-3">
        <TypeBadge type={item.item_type} app={item.target_app} />
        <div className="flex items-center gap-1 text-[10px] text-yellow-400">
          <Clock size={10} />
          <span>{item.retry_count || 0} retries</span>
        </div>
      </div>

      <p className="text-sm font-semibold text-gray-200 mb-1 truncate">
        {payload.metadata?.title || payload.front || payload.deck_name || item.item_id}
      </p>

      {(payload.target_path || payload.deck_name) && (
        <p className="text-[11px] text-gray-600 truncate mb-3">
          {payload.target_path || payload.deck_name}
        </p>
      )}

      {item.retry_count >= 10 && (
        <div className="flex items-center gap-1.5 text-[10px] text-red-400 mb-3">
          <AlertTriangle size={10} />
          Max retries reached — manual intervention required
        </div>
      )}

      <div className="flex gap-2">
        <button
          onClick={() => onRetry(item.item_id)}
          disabled={retrying || item.retry_count >= 10}
          className="flex-1 flex items-center justify-center gap-1.5 py-2 rounded-xl bg-yellow-500/20 border border-yellow-500/30 text-yellow-400 hover:bg-yellow-500/30 transition-all text-xs font-bold disabled:opacity-50"
        >
          {retrying ? <Loader2 size={12} className="animate-spin" /> : <RotateCcw size={12} />}
          Retry
        </button>
        <button
          onClick={() => onDismiss(item.item_id)}
          className="px-3 py-2 rounded-xl bg-gray-800/60 border border-gray-700/50 text-gray-500 hover:text-red-400 hover:border-red-500/30 transition-all"
        >
          <XCircle size={14} />
        </button>
      </div>
    </div>
  );
}

// ─── Main Component ───────────────────────────────────────────────────────────
export default function StagingAreaView() {
  const { API_BASE, lastEvent, user } = useWisdom();

  const [queue, setQueue] = useState([]);
  const [loading, setLoading] = useState(true);
  const [retryingAll, setRetryingAll] = useState(false);
  const [retryingItem, setRetryingItem] = useState(null);
  const [lastRetryResult, setLastRetryResult] = useState(null);

  // ─── Fetch Queue ────────────────────────────────────────────────────────
  const fetchQueue = useCallback(async () => {
    setLoading(true);
    try {
      const userId = user?.ldap || 'default';
      const res = await fetch(`${API_BASE}/api/v1/integrations/queue?user_id=${userId}`);
      if (res.ok) {
        const data = await res.json();
        setQueue(data.items || []);
      }
    } catch (err) {
      console.error('Failed to fetch queue:', err);
    }
    finally { setLoading(false); }
  }, [API_BASE, user]);

  useEffect(() => {
    Promise.resolve().then(() => fetchQueue());
    const interval = setInterval(fetchQueue, 30_000);
    return () => clearInterval(interval);
  }, [fetchQueue]);

  // ─── Live Sync Events via WebSocket ──────────────────────────────────
  useEffect(() => {
    if (!lastEvent) return;
    if (lastEvent.type === 'wisdom.integrations.sync_ready') {
      Promise.resolve().then(() => fetchQueue());
    }
    if (lastEvent.type === 'wisdom.integrations.item_synced') {
      Promise.resolve().then(() => {
        setQueue(prev => prev.filter(i => i.item_id !== lastEvent.item_id));
      });
    }
  }, [lastEvent, fetchQueue]);

  // ─── Retry All ──────────────────────────────────────────────────────
  const retryAll = async () => {
    setRetryingAll(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/integrations/retry`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: user?.ldap || 'default' }),
      });
      if (res.ok) {
        const result = await res.json();
        setLastRetryResult(result);
        setTimeout(fetchQueue, 2000);
      }
    } catch (err) {
      console.error('Failed to retry all items:', err);
    }
    finally { setRetryingAll(false); }
  };

  // ─── Retry Single ────────────────────────────────────────────────────
  const retryItem = async (itemId) => {
    setRetryingItem(itemId);
    try {
      await fetch(`${API_BASE}/api/v1/integrations/retry/${itemId}`, { method: 'POST' });
      setTimeout(fetchQueue, 1500);
    } catch (err) {
      console.error(`Failed to retry item ${itemId}:`, err);
    }
    finally { setRetryingItem(null); }
  };

  // ─── Dismiss ─────────────────────────────────────────────────────────
  const dismissItem = async (itemId) => {
    setQueue(prev => prev.filter(i => i.item_id !== itemId));
    try {
      await fetch(`${API_BASE}/api/v1/integrations/queue/${itemId}`, { method: 'DELETE' });
    } catch (err) {
      console.error(`Failed to dismiss item ${itemId}:`, err);
    }
  };

  const criticalItems = queue.filter(i => i.retry_count >= 10);
  const normalItems = queue.filter(i => i.retry_count < 10);

  return (
    <div className="h-full overflow-y-auto bg-[#0d1117] text-gray-100 p-6 space-y-6">

      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-black text-white tracking-tight flex items-center gap-3">
            <CloudOff className="text-yellow-400" size={22} />
            Knowledge Staging Area
          </h1>
          <p className="text-sm text-gray-500 mt-0.5">Items awaiting sync to Obsidian & Anki</p>
        </div>
        <div className="flex items-center gap-3">
          <button onClick={fetchQueue}
            className="flex items-center gap-2 px-4 py-2 rounded-xl bg-gray-800/60 border border-gray-700/50 text-gray-400 hover:text-white transition-all text-xs font-bold">
            <RefreshCw size={13} />
          </button>
          <button
            onClick={retryAll}
            disabled={retryingAll || queue.length === 0}
            className="flex items-center gap-2 px-4 py-2 rounded-xl bg-yellow-500/20 border border-yellow-500/30 text-yellow-400 hover:bg-yellow-500/30 transition-all text-xs font-bold disabled:opacity-50"
          >
            {retryingAll ? <Loader2 size={13} className="animate-spin" /> : <RotateCcw size={13} />}
            Retry All ({queue.length})
          </button>
        </div>
      </div>

      {/* Last Retry Result */}
      {lastRetryResult && (
        <div className="p-4 rounded-2xl border border-emerald-500/20 bg-emerald-500/5">
          <div className="flex items-center gap-3 text-sm">
            <CheckCircle2 className="text-emerald-400" size={16} />
            <span className="text-gray-300">
              Last retry: <span className="text-emerald-400 font-bold">{lastRetryResult.succeeded} synced</span>
              {lastRetryResult.failed > 0 && <span className="text-red-400 font-bold"> · {lastRetryResult.failed} failed</span>}
              {lastRetryResult.still_pending > 0 && <span className="text-yellow-400"> · {lastRetryResult.still_pending} still pending</span>}
            </span>
          </div>
        </div>
      )}

      {/* Connection Hint */}
      <div className="p-4 rounded-2xl border border-gray-800/40 bg-gray-900/20">
        <div className="flex items-center gap-3">
          <Wifi size={16} className="text-gray-500" />
          <div>
            <p className="text-xs font-bold text-gray-400">Why are items staged?</p>
            <p className="text-[11px] text-gray-600 mt-0.5">
              Items land here when Obsidian or Anki are offline. They auto-retry every 5 minutes when your apps are open.
            </p>
          </div>
        </div>
      </div>

      {/* Loading */}
      {loading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 size={28} className="text-indigo-500 animate-spin" />
        </div>
      ) : queue.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-gray-600 rounded-2xl border border-gray-800/30 bg-gray-900/10">
          <CheckCircle2 size={40} className="mb-4 text-emerald-500/40" />
          <p className="text-base font-bold text-gray-400">All clear!</p>
          <p className="text-sm text-gray-600 mt-1">Everything is synced to Obsidian & Anki</p>
        </div>
      ) : (
        <>
          {/* Critical items (max retries reached) */}
          {criticalItems.length > 0 && (
            <div>
              <div className="flex items-center gap-2 mb-3">
                <AlertTriangle size={14} className="text-red-400" />
                <h2 className="text-sm font-black text-red-400 uppercase tracking-widest">Needs Attention ({criticalItems.length})</h2>
              </div>
              <div className="grid grid-cols-2 gap-3">
                {criticalItems.map(item => (
                  <PendingItemCard key={item.item_id} item={item}
                    onRetry={retryItem} onDismiss={dismissItem}
                    retrying={retryingItem === item.item_id} />
                ))}
              </div>
            </div>
          )}

          {/* Normal pending */}
          {normalItems.length > 0 && (
            <div>
              <h2 className="text-sm font-black text-gray-300 uppercase tracking-widest mb-3">
                Pending Sync ({normalItems.length})
              </h2>
              <div className="grid grid-cols-2 gap-3">
                {normalItems.map(item => (
                  <PendingItemCard key={item.item_id} item={item}
                    onRetry={retryItem} onDismiss={dismissItem}
                    retrying={retryingItem === item.item_id} />
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
