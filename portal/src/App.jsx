import React, { useState } from 'react';
import { 
  Brain, 
  Shield, 
  Activity, 
  Network, 
  User, 
  Settings, 
  LogOut, 
  MessageSquare, 
  FileText, 
  Sparkles,
  Loader2,
  AlertCircle,
  MonitorDot,
  Rss,
  CloudOff,
  BookOpen
} from 'lucide-react';
import { WisdomProvider, useWisdom } from './context/WisdomContext';
import GraphView from './components/GraphView';
import MetabolismView from './components/MetabolismView';
import ChatView from './components/ChatView';
import NoteEditor from './components/NoteEditor';
import ReviewView from './components/ReviewView';
import LearningView from './components/LearningView';
import MissionControlView from './components/MissionControlView';
import ResearcherView from './components/ResearcherView';
import StagingAreaView from './components/StagingAreaView';
import StudyView from './components/StudyView';
import ExpertRegistry from './components/ExpertRegistry';

function AppContent() {
  const { 
    view, setView, 
    rigor, setRigor, 
    activeNamespace, setActiveNamespace, 
    namespaces,
    user, 
    loading, setLoading,
    error, API_BASE
  } = useWisdom();
  const [editingNode, setEditingNode] = useState(null);

  const handleEditNode = (node) => {
    setEditingNode(node);
    setView('NOTES');
  };

  const handleNewNote = () => {
    setEditingNode(null);
    setView('NOTES');
  };

  return (
    <div className="flex h-screen bg-[#0d1117] text-gray-100 overflow-hidden font-sans">
      {/* Sidebar */}
      <div className="w-72 border-r border-gray-800/50 bg-black/40 backdrop-blur-2xl p-6 flex flex-col gap-8 shadow-2xl z-20 overflow-y-auto custom-scrollbar h-full shrink-0">
        {/* Wisdom Logo Unchanged */}
        <div className="flex items-center gap-3 px-2">
          <div className="p-2.5 bg-indigo-500/10 rounded-xl border border-indigo-500/20 shadow-[0_0_15px_rgba(99,102,241,0.1)]">
            <Brain className="text-indigo-400 w-8 h-8" />
          </div>
          <div className="flex flex-col">
            <h1 className="text-2xl font-black tracking-tighter uppercase italic text-white leading-none text-indigo-100">Wisdom</h1>
            <span className="text-[10px] font-bold text-indigo-400/80 tracking-[0.2em] uppercase">Neural Atlas</span>
          </div>
        </div>

        <nav className="space-y-1.5">
          <div className="px-3 text-[10px] font-black text-gray-500 uppercase tracking-[0.2em] mb-3">Core Systems</div>
          {[
            { id: 'GRAPH', label: 'Knowledge Graph', icon: <Network size={18} /> },
            { id: 'CHAT', label: 'Conversational', icon: <MessageSquare size={18} /> },
            { id: 'STUDY', label: 'Study Session', icon: <BookOpen size={18} /> },
            { id: 'REVIEW', label: 'Spaced Repetition', icon: <Sparkles size={18} /> },
            { id: 'NOTES', label: 'Note Repository', icon: <FileText size={18} /> },
            { id: 'METABOLISM', label: 'Metabolic Audit', icon: <Activity size={18} /> },
          ].map(item => (
            <button
              key={item.id}
              onClick={() => setView(item.id)}
              className={`w-full flex items-center gap-3.5 p-3 rounded-xl transition-all duration-300 group relative overflow-hidden ${     
                view === item.id
                  ? 'bg-indigo-600 text-white font-bold shadow-lg shadow-indigo-500/20'
                  : 'text-gray-400 hover:bg-gray-800/60 hover:text-gray-100'
              }`}
            >
              <span className={`transition-transform duration-300 ${view === item.id ? 'scale-110' : 'group-hover:scale-110 relative z-10'}`}>
                {item.icon}
              </span>
              <span className="text-sm tracking-tight relative z-10">{item.label}</span>
              {view === item.id && (
                <div className="absolute inset-0 bg-gradient-to-r from-white/10 to-transparent pointer-events-none" />
              )}
            </button>
          ))}

          <div className="px-3 text-[10px] font-black text-gray-500 uppercase tracking-[0.2em] mb-3 mt-5">Observability</div>
          {[
            { id: 'MISSION', label: 'Mission Control', icon: <MonitorDot size={18} /> },
            { id: 'EXPERTS', label: 'Expert Registry', icon: <Sparkles size={18} /> },
            { id: 'RESEARCHER', label: 'Researcher', icon: <Rss size={18} /> },
            { id: 'STAGING', label: 'Staging Area', icon: <CloudOff size={18} /> },
          ].map(item => (
            <button
              key={item.id}
              onClick={() => setView(item.id)}
              className={`w-full flex items-center gap-3.5 p-3 rounded-xl transition-all duration-300 group relative overflow-hidden ${
                view === item.id
                  ? 'bg-indigo-600 text-white font-bold shadow-lg shadow-indigo-500/20'
                  : 'text-gray-400 hover:bg-gray-800/60 hover:text-gray-100'
              }`}
            >
              <span className={`transition-transform duration-300 ${view === item.id ? 'scale-110' : 'group-hover:scale-110 relative z-10'}`}>
                {item.icon}
              </span>
              <span className="text-sm tracking-tight relative z-10">{item.label}</span>
              {view === item.id && (
                <div className="absolute inset-0 bg-gradient-to-r from-white/10 to-transparent pointer-events-none" />
              )}
            </button>
          ))}

          <button 
            onClick={handleNewNote}
            className="mt-4 w-full flex items-center justify-center gap-2 p-3 rounded-xl bg-indigo-500/20 text-indigo-300 border border-indigo-500/30 hover:bg-indigo-500/30 transition-all font-black uppercase tracking-wider text-xs shadow-[0_0_15px_rgba(99,102,241,0.1)]"
          >
            <FileText size={16} />
            Write New Note
          </button>
        </nav>

        <div className="flex flex-col gap-2 px-2">
          <button 
            onClick={async () => {
              const dir = prompt("Enter directory to map:", ".");
              if (dir) {
                setLoading(true);
                try {
                  const res = await fetch(`${API_BASE}/cortex/map`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ directory: dir })
                  });
                  if (res.ok) alert("Codebase mapping initiated");
                  else {
                    const err = await res.text();
                    alert(`Failed to initiate mapping: ${err}`);
                  }
                } catch (e) { 
                  alert(`Connection error: ${e.message}`); 
                } finally { 
                  setLoading(false); 
                }
              }
            }}
            className="p-3 bg-gray-800/40 border border-gray-700/50 rounded-xl text-[9px] font-black text-gray-400 uppercase tracking-widest hover:bg-indigo-500/10 hover:text-indigo-300 transition-all"
          >
            Map Codebase
          </button>
          
          <input 
            type="file" 
            id="doc-upload" 
            className="hidden" 
            accept=".pdf,.txt,.docx,.pptx,.xlsx,.png,.jpg,.jpeg"
            onChange={async (e) => {
              if (e.target.files[0]) {
                const formData = new FormData();
                formData.append('document', e.target.files[0]);
                setLoading(true);
                try {
                  const res = await fetch(`${API_BASE}/cortex/upload`, {
                    method: 'POST',
                    body: formData
                  });
                  if (res.ok) alert("Document ingested successfully");
                  else {
                    const err = await res.text();
                    alert(`Upload failed: ${err}`);
                  }
                } catch (e) { 
                  alert(`Connection error: ${e.message}`); 
                } finally { 
                  setLoading(false); 
                }
              }
            }}
          />
          
          <button 
            onClick={() => document.getElementById('doc-upload').click()}
            className="p-3 bg-gray-800/40 border border-gray-700/50 rounded-xl text-[9px] font-black text-gray-400 uppercase tracking-widest hover:bg-green-500/10 hover:text-green-300 transition-all"
          >
            Upload Knowledge (PDF/Img)
          </button>
        </div>

        <div className="space-y-6">
          <div>
            <div className="px-3 text-[10px] font-black text-gray-500 uppercase tracking-[0.2em] mb-3">Axiom Controls</div>
            <button 
              onClick={() => setRigor(rigor === 'LOW' ? 'HIGH' : 'LOW')}
              className={`w-full flex items-center justify-between p-3.5 rounded-xl border transition-all duration-500 group ${
                rigor === 'HIGH' 
                  ? 'border-red-500/40 bg-red-500/5 text-red-400 shadow-[inset_0_0_20px_rgba(239,68,68,0.05)]' 
                  : 'border-gray-800/80 bg-gray-900/40 text-gray-400 hover:border-gray-700 hover:bg-gray-800/40'
              }`}
            >
              <div className="flex items-center gap-3">
                <div className={`p-1.5 rounded-lg transition-colors duration-500 ${rigor === 'HIGH' ? 'bg-red-500/20' : 'bg-gray-800'}`}>
                  <Shield className={rigor === 'HIGH' ? 'text-red-500 animate-pulse' : 'text-gray-500'} size={18} />
                </div>
                <span className="text-sm font-semibold tracking-tight">Rigor: {rigor}</span>
              </div>
              <div className={`w-2.5 h-2.5 rounded-full transition-all duration-500 ${rigor === 'HIGH' ? 'bg-red-500 shadow-[0_0_12px_rgba(239,68,68,0.8)]' : 'bg-gray-700'}`} />
            </button>
          </div>

          <div className="space-y-2">
            <div className="px-3 text-[10px] font-black text-gray-500 uppercase tracking-[0.2em] mb-3">Thought Spaces</div>
            <div className="grid grid-cols-1 gap-1.5">
              {(namespaces && namespaces.length > 0 ? namespaces : ['ns-general']).map(ns => (
                <button 
                  key={ns}
                  onClick={() => setActiveNamespace(ns)}
                  className={`w-full text-left px-4 py-2.5 rounded-xl text-[11px] font-bold transition-all duration-300 border ${
                    activeNamespace === ns 
                      ? 'bg-indigo-500/10 text-indigo-400 border-indigo-500/30 shadow-[0_0_15px_rgba(99,102,241,0.05)]' 
                      : 'text-gray-500 border-transparent hover:bg-gray-800/40 hover:text-gray-300'
                  }`}
                >
                  {ns}
                </button>
              ))}
            </div>
          </div>
        </div>

        <div className="mt-auto">
          <div className="p-4 bg-gray-900/40 rounded-2xl border border-gray-800/60 hover:border-indigo-500/40 transition-all duration-500 group/user shadow-lg">
            <div className="flex items-center gap-3 mb-4">
              <div className="w-11 h-11 rounded-full bg-gradient-to-tr from-indigo-500 via-indigo-300 to-indigo-600 p-[1.5px] shadow-lg shadow-indigo-500/5 group-hover/user:scale-105 transition-transform duration-500">
                <div className="w-full h-full rounded-full bg-[#0d1117] flex items-center justify-center">
                  <User size={22} className="text-indigo-400" />
                </div>
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-sm font-black text-gray-100 truncate tracking-tight">{user?.ldap || 'Anonymous'}</div>
                <div className="text-[10px] text-indigo-400 font-black uppercase tracking-widest flex items-center gap-1.5">
                  <span className="w-1.5 h-1.5 rounded-full bg-indigo-500 animate-pulse" />
                  {user?.role || 'Guest'}
                </div>
              </div>
            </div>
            <div className="flex gap-2.5">
              <button className="flex-1 p-2.5 bg-gray-800/50 rounded-xl hover:bg-gray-700/80 hover:text-white transition-all duration-300 flex justify-center border border-gray-700/30">
                <Settings size={16} className="text-gray-400 group-hover:rotate-45 transition-transform duration-500" />
              </button>
              <button className="flex-1 p-2.5 bg-gray-800/50 rounded-xl hover:bg-red-500/10 hover:border-red-500/30 transition-all duration-300 flex justify-center border border-gray-700/30 group/logout">
                <LogOut size={16} className="text-gray-400 group-hover/logout:text-red-500 transition-colors" />
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 relative bg-[#0d1117]">
        {loading && (
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center">
            <div className="flex flex-col items-center gap-4">
              <Loader2 className="text-indigo-500 animate-spin" size={48} />
              <p className="text-indigo-300 font-black uppercase tracking-[0.2em] animate-pulse">Syncing Cortex...</p>
            </div>
          </div>
        )}

        {error && (
          <div className="absolute top-6 left-1/2 -translate-x-1/2 z-50 bg-red-500/10 border border-red-500/20 p-4 rounded-2xl flex items-center gap-4 backdrop-blur-xl animate-in fade-in slide-in-from-top-4">
            <AlertCircle className="text-red-400" size={20} />
            <span className="text-sm font-bold text-red-200">{error}</span>
          </div>
        )}

        {view === 'GRAPH' && (
          <div className="w-full h-full">
            <GraphView 
              namespace={activeNamespace} 
              minWeight={rigor === 'HIGH' ? 0.6 : 0.0} 
              onEditNode={handleEditNode}
            />
          </div>
        )}
        {view === 'METABOLISM' && (
          <div className="h-full overflow-hidden">
            <MetabolismView />
          </div>
        )}
        {view === 'LEARNING' && (
          <div className="h-full overflow-hidden">
            <LearningView />
          </div>
        )}
        {view === 'CHAT' && (
          <div className="h-full overflow-hidden">
            <ChatView onDistill={handleEditNode} />
          </div>
        )}
        {view === 'REVIEW' && (
          <div className="h-full overflow-hidden">
            <ReviewView />
          </div>
        )}
        {view === 'NOTES' && (
          <div className="h-full overflow-hidden">
            <NoteEditor 
              initialNode={editingNode} 
              onBack={() => setView('GRAPH')} 
            />
          </div>
        )}
        {view === 'STUDY' && (
          <div className="h-full overflow-hidden">
            <StudyView />
          </div>
        )}
        {view === 'MISSION' && (
          <div className="h-full overflow-hidden">
            <MissionControlView />
          </div>
        )}
        {view === 'RESEARCHER' && (
          <div className="h-full overflow-hidden">
            <ResearcherView />
          </div>
        )}
        {view === 'STAGING' && (
          <div className="h-full overflow-hidden">
            <StagingAreaView />
          </div>
        )}
        {view === 'EXPERTS' && (
          <div className="h-full overflow-hidden">
            <ExpertRegistry />
          </div>
        )}
      </div>
    </div>
  );
}

function App() {
  return (
    <WisdomProvider>
      <AppContent />
    </WisdomProvider>
  );
}

export default App;

