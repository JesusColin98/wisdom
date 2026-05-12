import React, { useState, useEffect, useCallback, useRef } from 'react';
import ForceGraph2D from 'react-force-graph-2d';
import { Network, Search, Info, Layers, Edit3, Flame, Snowflake, Star } from 'lucide-react';
import { useWisdom } from '../context/WisdomContext';

const CLASS_COLORS = {
  'PERSON': '#a855f7',
  'CONCEPT': '#3b82f6',
  'ERROR_PATTERN': '#ef4444',
  'ROLE': '#f59e0b',
  'PATTERN': '#22c55e',
  'default': '#64748b'
};

const GraphView = ({ namespace, onEditNode }) => {
  const { API_BASE, setLoading, setError, lastEvent } = useWisdom();
  const [data, setData] = useState({ nodes: [], links: [] });
  const [internalLoading, setInternalLoading] = useState(true);
  const [selectedNode, setSelectedNode] = useState(null);
  const graphRef = useRef();

  const fetchGraphData = useCallback(async () => {
    setInternalLoading(true);
    setLoading(true);
    setError(null);
    try {
      const [nodesRes, edgesRes] = await Promise.all([
        fetch(`${API_BASE}/cortex/nodes?namespace=${namespace}`),
        fetch(`${API_BASE}/cortex/edges`)
      ]);

      if (!nodesRes.ok || !edgesRes.ok) throw new Error("Failed to sync neural nodes");

      const nodesData = await nodesRes.json();
      const edgesData = await edgesRes.json();

      const nodes = Array.isArray(nodesData) ? nodesData.filter(Boolean) : [];
      const edges = Array.isArray(edgesData) ? edgesData.filter(Boolean) : [];

      // Format for react-force-graph
      const formattedNodes = (nodes || []).map(n => ({
        id: n.id,
        name: n.id,
        content: n.content,
        author: n.author,
        stratum: n.stratum,
        entityClass: n.entity_class,
        impact: n.impact_score || 0,
        val: n.stratum === 'HOT' ? 14 : 8,
        color: CLASS_COLORS[n.entity_class] || CLASS_COLORS.default,
        isHighImpact: (n.impact_score || 0) > 0.8
      }));

      const formattedLinks = (edges || []).map(e => ({
        source: e.source_id,
        target: e.target_id,
        label: e.relation_type,
        color: '#334155'
      }));

      setData({ nodes: formattedNodes, links: formattedLinks });
    } catch (error) {
      console.error("Failed to fetch graph data:", error);
      setError(error.message);
    } finally {
      setInternalLoading(false);
      setLoading(false);
    }
  }, [namespace, API_BASE, setLoading, setError]);

  useEffect(() => {
    let mounted = true;
    const load = async () => {
        if (mounted) await fetchGraphData();
    };
    load();
    return () => { mounted = false; };
  }, [fetchGraphData]);

  useEffect(() => {
    if (lastEvent && (lastEvent.type === 'REM_CONSOLIDATED' || lastEvent.type === 'MAPPED')) {
        fetchGraphData(); // eslint-disable-line react-hooks/set-state-in-effect
    }
  }, [lastEvent, fetchGraphData]);

  const handleNodeClick = node => {
    setSelectedNode(node);
    if (graphRef.current) {
        graphRef.current.centerAt(node.x, node.y, 1000);
        graphRef.current.zoom(3, 1000);
    }
  };

  const handleLineage = async (direction) => {
    if (!selectedNode) return;
    setLoading(true);
    try {
        const res = await fetch(`${API_BASE}/cortex/lineage?id=${selectedNode.id}&direction=${direction}`);
        if (res.ok) {
            const lineageNodes = await res.json();
            window.alert(`Found ${lineageNodes.length} nodes in lineage ${direction}`);
        }
    } catch (e) { setError(e.message); }
    finally { setLoading(false); }
  };

  return (
    <div className="relative w-full h-full bg-[#0d1117]">
      {internalLoading && (
        <div className="absolute inset-0 flex items-center justify-center z-50 bg-black/20 backdrop-blur-sm">
          <div className="flex flex-col items-center gap-4">
            <Network className="text-indigo-500 animate-pulse" size={48} />
            <span className="text-xs font-black text-indigo-300 uppercase tracking-widest">Reconstructing Cortex...</span>
          </div>
        </div>
      )}


      <ForceGraph2D
        ref={graphRef}
        graphData={data}
        nodeLabel="id"
        linkDirectionalArrowLength={3.5}
        linkDirectionalArrowRelPos={1}
        linkCurvature={0.25}
        backgroundColor="#0d1117"
        onNodeClick={handleNodeClick}
        nodeCanvasObject={(node, ctx, globalScale) => {
          const label = node.id;
          const fontSize = 12/globalScale;
          ctx.font = `${fontSize}px Inter, sans-serif`;
          const textWidth = ctx.measureText(label).width;
          const bckgDimensions = [textWidth, fontSize].map(n => n + fontSize * 0.4); 

          // Dopamine Glow for high impact
          if (node.isHighImpact) {
            ctx.shadowColor = node.color;
            ctx.shadowBlur = 15 / globalScale;
          } else {
            ctx.shadowBlur = 0;
          }

          ctx.fillStyle = 'rgba(13, 17, 23, 0.9)';
          ctx.beginPath();
          ctx.roundRect(node.x - bckgDimensions[0] / 2, node.y - bckgDimensions[1] / 2, bckgDimensions[0], bckgDimensions[1], 4/globalScale);
          ctx.fill();
          
          ctx.strokeStyle = node.color;
          ctx.lineWidth = (node.stratum === 'HOT' ? 2 : 1) / globalScale;
          ctx.stroke();

          // Reset shadow for text
          ctx.shadowBlur = 0;

          ctx.textAlign = 'center';
          ctx.textBaseline = 'middle';
          ctx.fillStyle = '#e2e8f0';
          ctx.fillText(label, node.x, node.y);

          node.__bckgDimensions = bckgDimensions;
        }}
      />

      {selectedNode && (
        <div className="absolute right-8 top-24 w-96 bg-black/60 backdrop-blur-2xl border border-white/10 rounded-3xl p-8 shadow-[0_40px_100px_rgba(0,0,0,0.7)] animate-in fade-in slide-in-from-right-8 duration-500 z-30">
          <div className="flex justify-between items-start mb-6">
            <div className="flex gap-2">
              <div className="p-3 bg-indigo-500/10 rounded-2xl border border-indigo-500/20">
                <Info className="text-indigo-400" size={24} />
              </div>
              <button 
                onClick={() => onEditNode(selectedNode)}
                className="p-2.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl transition-all shadow-lg flex items-center gap-2 text-[10px] font-black uppercase tracking-widest"
              >
                <Edit3 size={14} /> Edit Node
              </button>
            </div>
            <button 
              onClick={() => setSelectedNode(null)}
              className="text-gray-500 hover:text-white transition-colors"
            >
              ✕
            </button>
          </div>
          
          <div className="space-y-6">
            <div>
              <div className="flex items-center gap-2 mb-1">
                <span className="text-[10px] font-black text-gray-500 uppercase tracking-widest">Node Identifier</span>
                {selectedNode.stratum === 'HOT' ? (
                    <span className="flex items-center gap-1 text-[8px] font-black bg-orange-500/20 text-orange-400 px-2 py-0.5 rounded-full border border-orange-500/20 animate-pulse">
                        <Flame size={8} /> HOT STRATUM
                    </span>
                ) : (
                    <span className="flex items-center gap-1 text-[8px] font-black bg-blue-500/20 text-blue-400 px-2 py-0.5 rounded-full border border-blue-500/20">
                        <Snowflake size={8} /> COLD STRATUM
                    </span>
                )}
                {selectedNode.isHighImpact && (
                    <span className="flex items-center gap-1 text-[8px] font-black bg-yellow-500/20 text-yellow-400 px-2 py-0.5 rounded-full border border-yellow-500/20">
                        <Star size={8} fill="currentColor" /> HIGH IMPACT
                    </span>
                )}
              </div>
              <h3 className="text-xl font-bold text-white">{selectedNode.id}</h3>
              <div className="flex gap-3 mt-1">
                <p className="text-[10px] text-indigo-400 font-bold uppercase">Class: {selectedNode.entityClass || 'GENERAL'}</p>
                <p className="text-[10px] text-gray-500 font-bold uppercase">Author: {selectedNode.author || 'system'}</p>
              </div>
              
              <div className="flex gap-2 mt-4">
                <button 
                    onClick={() => handleLineage('UP')}
                    className="flex-1 py-2 bg-gray-800 border border-gray-700 rounded-xl text-[9px] font-black uppercase hover:bg-indigo-500/10 hover:border-indigo-500/30 transition-all"
                >
                    Zoom Out (UP)
                </button>
                <button 
                    onClick={() => handleLineage('DOWN')}
                    className="flex-1 py-2 bg-gray-800 border border-gray-700 rounded-xl text-[9px] font-black uppercase hover:bg-indigo-500/10 hover:border-indigo-500/30 transition-all"
                >
                    Drill Down (DOWN)
                </button>
              </div>
            </div>

            <div>
              <span className="text-[10px] font-black text-gray-500 uppercase tracking-widest">Grounded Knowledge</span>
              <div className="mt-3 p-5 bg-gray-900/80 rounded-2xl border border-gray-800 text-sm leading-relaxed text-gray-200 max-h-64 overflow-y-auto custom-scrollbar font-serif">
                {selectedNode.content}
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="p-4 bg-indigo-500/5 rounded-2xl border border-indigo-500/10">
                <span className="text-[10px] font-black text-gray-500 uppercase tracking-widest block mb-1">Impact Score</span>
                <span className="text-sm font-bold text-indigo-400">{(selectedNode.impact * 100).toFixed(0)}%</span>
              </div>
              <div className="p-4 bg-green-500/5 rounded-2xl border border-green-500/10">
                <span className="text-[10px] font-black text-gray-500 uppercase tracking-widest block mb-1">Verification</span>
                <span className="text-sm font-bold text-green-400 font-mono">TRUSTED</span>
              </div>
            </div>
          </div>
        </div>
      )}

      <div className="absolute left-8 bottom-8 flex flex-col gap-3 z-30">
        <button 
          onClick={fetchGraphData}
          className="p-4 bg-black/60 backdrop-blur-xl border border-white/10 rounded-2xl text-gray-400 hover:text-indigo-400 transition-all shadow-2xl hover:scale-110 active:scale-95 group"
          title="Resync Graph"
        >
          <Search size={20} className="group-hover:rotate-90 transition-transform duration-500" />
        </button>
        <button 
          onClick={() => setShowExplorer(!showExplorer)}
          className={`p-4 backdrop-blur-xl border rounded-2xl transition-all shadow-2xl hover:scale-110 active:scale-95 ${
            showExplorer ? 'bg-indigo-600/80 border-indigo-400 text-white' : 'bg-black/60 border-white/10 text-gray-400 hover:text-indigo-400'
          }`}
          title="Data Explorer"
        >
          <List size={20} />
        </button>
        <button className="p-4 bg-black/60 backdrop-blur-xl border border-white/10 rounded-2xl text-gray-400 hover:text-indigo-400 transition-all shadow-2xl hover:scale-110 active:scale-95">
          <Layers size={20} />
        </button>
      </div>

      {showExplorer && (
        <div className="absolute left-28 bottom-8 top-8 w-80 bg-black/80 backdrop-blur-2xl border border-white/10 rounded-3xl p-6 shadow-[0_40px_100px_rgba(0,0,0,0.7)] animate-in fade-in slide-in-from-left-8 duration-500 z-30 flex flex-col">
          <div className="flex items-center gap-3 mb-6">
            <div className="p-2.5 bg-indigo-500/10 rounded-xl border border-indigo-500/20">
              <Search className="text-indigo-400" size={20} />
            </div>
            <input
              type="text"
              placeholder="Explore notes & entities..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="bg-transparent border-b border-gray-700 pb-1 text-sm font-bold text-white placeholder:text-gray-600 w-full focus:outline-none focus:border-indigo-500 transition-colors"
            />
          </div>
          <div className="flex-1 overflow-y-auto custom-scrollbar space-y-2 pr-2">
            {data.nodes
              .filter(n => n.id.toLowerCase().includes(searchTerm.toLowerCase()) || (n.content && n.content.toLowerCase().includes(searchTerm.toLowerCase())))
              .map(node => (
                <button
                  key={node.id}
                  onClick={() => {
                    setSelectedNode(node);
                    if (graphRef.current) {
                      graphRef.current.centerAt(node.x, node.y, 1000);
                      graphRef.current.zoom(2, 1000);
                    }
                  }}
                  className="w-full text-left p-3 rounded-xl bg-gray-900/50 hover:bg-indigo-500/10 border border-transparent hover:border-indigo-500/30 transition-all group flex items-center justify-between"
                >
                  <div className="overflow-hidden">
                    <div className="text-sm font-bold text-gray-200 truncate">{node.id}</div>
                    <div className="text-[10px] font-black text-gray-600 uppercase tracking-widest mt-1 truncate">
                      {node.entityClass || 'NOTE'} • {node.stratum}
                    </div>
                  </div>
                  <ChevronRight size={14} className="text-gray-600 group-hover:text-indigo-400 transition-colors shrink-0" />
                </button>
            ))}
            {data.nodes.length === 0 && (
              <div className="text-center text-gray-500 text-sm py-8 font-bold">No nodes found</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default GraphView;
