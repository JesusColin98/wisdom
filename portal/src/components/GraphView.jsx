import React, { useState, useEffect, useCallback, useRef } from 'react';
import ForceGraph2D from 'react-force-graph-2d';
import { Network, Search, Info, Layers, Edit3 } from 'lucide-react';
import { useWisdom } from '../context/WisdomContext';

const GraphView = ({ namespace, onEditNode }) => {
  const { API_BASE, setLoading, setError } = useWisdom();
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

      const nodes = await nodesRes.json();
      const edges = await edgesRes.json();

      // Format for react-force-graph
      const formattedNodes = nodes.map(n => ({
        id: n.id,
        name: n.id,
        content: n.content,
        author: n.author,
        val: 10,
        color: n.id.includes('err') ? '#ef4444' : '#6366f1'
      }));

      const formattedLinks = edges.map(e => ({
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
    fetchGraphData();
  }, [fetchGraphData]);

  const handleNodeClick = node => {
    setSelectedNode(node);
    if (graphRef.current) {
        graphRef.current.centerAt(node.x, node.y, 1000);
        graphRef.current.zoom(3, 1000);
    }
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
        nodeAutoColorBy="group"
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

          ctx.fillStyle = 'rgba(13, 17, 23, 0.9)';
          ctx.beginPath();
          ctx.roundRect(node.x - bckgDimensions[0] / 2, node.y - bckgDimensions[1] / 2, bckgDimensions[0], bckgDimensions[1], 4/globalScale);
          ctx.fill();
          
          ctx.strokeStyle = node.color;
          ctx.lineWidth = 1/globalScale;
          ctx.stroke();

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
              <span className="text-[10px] font-black text-gray-500 uppercase tracking-widest">Node Identifier</span>
              <h3 className="text-xl font-bold text-white mt-1">{selectedNode.id}</h3>
              <p className="text-[10px] text-indigo-400 font-bold mt-1 uppercase">Author: {selectedNode.author || 'system'}</p>
            </div>

            <div>
              <span className="text-[10px] font-black text-gray-500 uppercase tracking-widest">Grounded Knowledge</span>
              <div className="mt-3 p-5 bg-gray-900/80 rounded-2xl border border-gray-800 text-sm leading-relaxed text-gray-200 max-h-64 overflow-y-auto custom-scrollbar font-serif">
                {selectedNode.content}
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="p-4 bg-indigo-500/5 rounded-2xl border border-indigo-500/10">
                <span className="text-[10px] font-black text-gray-500 uppercase tracking-widest block mb-1">Centrality</span>
                <span className="text-sm font-bold text-indigo-400">PPR: 0.85</span>
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
        <button className="p-4 bg-black/60 backdrop-blur-xl border border-white/10 rounded-2xl text-gray-400 hover:text-indigo-400 transition-all shadow-2xl hover:scale-110 active:scale-95">
          <Layers size={20} />
        </button>
      </div>
    </div>
  );
};

export default GraphView;
