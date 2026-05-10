import React, { useState, useEffect } from 'react';
import CodeMirror from '@uiw/react-codemirror';
import { markdown, markdownLanguage } from '@codemirror/lang-markdown';
import { languages } from '@codemirror/language-data';
import { oneDark } from '@codemirror/theme-one-dark';
import { Save, FileText, Link, Sparkles, ArrowLeft, CheckCircle } from 'lucide-react';
import { useWisdom } from '../context/WisdomContext';

const NoteEditor = ({ initialNode, onBack }) => {
  const { API_BASE, user } = useWisdom();
  const [content, setContent] = useState(initialNode?.content || '# New Wisdom Note\n\nType your SRE observations here. Use [[Wiki-Links]] to connect ideas.');
  const [id, setId] = useState(initialNode?.id || '');
  const [isSaving, setIsSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState(null);

  useEffect(() => {
    if (initialNode) {
      const timer = setTimeout(() => {
        setContent(prev => {
            if (prev !== initialNode.content) return initialNode.content;
            return prev;
        });
        setId(prev => {
            if (prev !== initialNode.id) return initialNode.id;
            return prev;
        });
      }, 0);
      return () => clearTimeout(timer);
    }
  }, [initialNode]);

  const handleSave = async () => {
    setIsSaving(true);
    setSaveStatus(null);
    try {
      const response = await fetch(`${API_BASE}/cortex/notes`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          id: id || `note-${Date.now()}`,
          content: content,
          author: user.ldap,
          namespace_id: 'ns-engineering'
        })
      });

      if (!response.ok) throw new Error("Save failed");
      
      setSaveStatus('SUCCESS');
      setTimeout(() => setSaveStatus(null), 3000);
    } catch (error) {
      console.error("Save error:", error);
      setSaveStatus('ERROR');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="flex h-full flex-col bg-[#0d1117] text-gray-200">
      {/* Toolbar */}
      <div className="p-4 border-b border-gray-800 bg-black/40 flex items-center justify-between sticky top-0 z-10 backdrop-blur-xl">
        <div className="flex items-center gap-6">
          <button 
            onClick={onBack}
            className="p-2 hover:bg-gray-800 rounded-xl transition-colors text-gray-400 hover:text-white"
          >
            <ArrowLeft size={20} />
          </button>
          <div className="flex items-center gap-3">
            <div className="p-2 bg-indigo-500/10 rounded-lg border border-indigo-500/20">
              <FileText className="text-indigo-400" size={18} />
            </div>
            <input 
              type="text" 
              value={id}
              onChange={(e) => setId(e.target.value)}
              placeholder="note-identifier"
              className="bg-transparent border-none focus:ring-0 text-sm font-black tracking-tight text-white w-64 placeholder:text-gray-700"
            />
          </div>
        </div>

        <div className="flex items-center gap-3">
          {saveStatus === 'SUCCESS' && (
            <div className="flex items-center gap-2 px-3 py-1.5 bg-green-500/10 border border-green-500/20 rounded-lg text-green-400 text-[10px] font-black uppercase tracking-widest animate-in fade-in zoom-in duration-300">
              <CheckCircle size={12} />
              Cortex Updated
            </div>
          )}
          <button 
            onClick={handleSave}
            disabled={isSaving}
            className={`flex items-center gap-2.5 px-5 py-2 rounded-xl text-xs font-black transition-all shadow-xl active:scale-95 ${
              isSaving 
                ? 'bg-gray-800 text-gray-500' 
                : 'bg-indigo-600 hover:bg-indigo-500 text-white shadow-indigo-500/20'
            }`}
          >
            <Save size={16} />
            {isSaving ? 'PERSISTING...' : 'COMMIT TO CORTEX'}
          </button>
        </div>
      </div>

      {/* Editor Area */}
      <div className="flex-1 flex overflow-hidden">
        {/* CodeMirror Editor */}
        <div className="flex-1 border-r border-gray-800 overflow-y-auto custom-scrollbar">
          <CodeMirror
            value={content}
            height="100%"
            theme={oneDark}
            extensions={[markdown({ base: markdownLanguage, codeLanguages: languages })]}
            onChange={(value) => setContent(value)}
            className="text-sm font-mono h-full"
            basicSetup={{
              lineNumbers: true,
              foldGutter: true,
              dropCursor: true,
              allowMultipleSelections: true,
              indentOnInput: true,
            }}
          />
        </div>

        {/* Live Preview (Simple Render) */}
        <div className="flex-1 bg-black/20 p-12 overflow-y-auto custom-scrollbar prose prose-invert prose-indigo max-w-none">
          <div className="flex items-center gap-2 text-[10px] font-black text-gray-500 uppercase tracking-[0.2em] mb-8">
            <Sparkles size={12} />
            Wisdom Live Preview
          </div>
          <div className="markdown-preview font-serif leading-relaxed text-gray-300 whitespace-pre-wrap">
            {content || "Nothing to preview..."}
          </div>
        </div>
      </div>
      
      {/* Bottom Info */}
      <div className="p-3 bg-gray-900/40 border-t border-gray-800 flex justify-between items-center text-[9px] font-black text-gray-600 uppercase tracking-widest px-8">
        <div className="flex items-center gap-4">
          <span className="flex items-center gap-1.5"><Link size={10} /> Link-aware editing active</span>
          <span className="w-1 h-1 rounded-full bg-gray-700" />
          <span>Markdown GFM enabled</span>
        </div>
        <div>
          {user.ldap}@google.com • {user.is_admin ? 'L7_ADMIN' : 'STANDARD_USER'}
        </div>
      </div>
    </div>
  );
};

export default NoteEditor;
