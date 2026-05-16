import React, { useState } from 'react';
import { 
  Compass, 
  FileText, 
  Search, 
  Video,
  Loader2, 
  CheckCircle2, 
  Circle, 
  ArrowRight,
  Sparkles,
  BookOpen,
  Target
} from 'lucide-react';
import { useWisdom } from '../context/WisdomContext';

const LearningView = () => {
  const { API_BASE, user } = useWisdom();
  const [activeTab, setActiveTab] = useState('topic');
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState({
    topic: '',
    url: '',
    content: ''
  });
  const [path, setPath] = useState(null);

  const generatePath = async () => {
    setLoading(true);
    try {
      const response = await fetch(`${API_BASE}/learning/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          type: activeTab,
          topic: formData.topic,
          url: formData.url,
          content: formData.content,
          user_id: user?.ldap || 'anonymous'
        })
      });
      const data = await response.json();
      setPath(data);
    } catch (error) {
      console.error("Failed to generate learning path:", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col h-full bg-[#0a0a0b] text-gray-100 overflow-hidden">
      {/* Header */}
      <div className="p-6 border-b border-gray-800 bg-[#0f0f12]">
        <div className="flex items-center gap-3 mb-2">
          <div className="p-2 bg-indigo-500/10 rounded-lg">
            <Compass className="w-6 h-6 text-indigo-400" />
          </div>
          <div>
            <h1 className="text-xl font-semibold bg-gradient-to-r from-white to-gray-400 bg-clip-text text-transparent">
              Proactive Learning Engine
            </h1>
            <p className="text-sm text-gray-500">Autonomous roadmap generation for any subject</p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6 space-y-8 max-w-4xl mx-auto w-full">
        {/* Input Section */}
        {!path && (
          <div className="space-y-6 animate-in fade-in duration-500">
            <div className="flex p-1 bg-[#16161a] rounded-xl border border-gray-800 w-fit">
              {[
                { id: 'topic', label: 'Topic Search', icon: Search },
                { id: 'youtube', label: 'YouTube Video', icon: Youtube },
                { id: 'document', label: 'Text Summary', icon: FileText }
              ].map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                    activeTab === tab.id 
                    ? 'bg-indigo-600 text-white shadow-lg' 
                    : 'text-gray-400 hover:text-gray-200'
                  }`}
                >
                  <tab.icon className="w-4 h-4" />
                  {tab.label}
                </button>
              ))}
            </div>

            <div className="bg-[#16161a] p-6 rounded-2xl border border-gray-800 shadow-2xl space-y-4">
              {activeTab === 'topic' && (
                <div className="space-y-2">
                  <label className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Learning Goal</label>
                  <input
                    type="text"
                    placeholder="e.g. History of the French Revolution, Advanced Chess Theory..."
                    className="w-full bg-[#0a0a0b] border border-gray-800 rounded-xl px-4 py-3 focus:outline-none focus:border-indigo-500 transition-colors text-gray-200"
                    value={formData.topic}
                    onChange={(e) => setFormData({...formData, topic: e.target.value})}
                  />
                </div>
              )}

              {activeTab === 'youtube' && (
                <div className="space-y-2">
                  <label className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Video URL</label>
                  <input
                    type="text"
                    placeholder="https://www.youtube.com/watch?v=..."
                    className="w-full bg-[#0a0a0b] border border-gray-800 rounded-xl px-4 py-3 focus:outline-none focus:border-indigo-500 transition-colors text-gray-200"
                    value={formData.url}
                    onChange={(e) => setFormData({...formData, url: e.target.value})}
                  />
                </div>
              )}

              {activeTab === 'document' && (
                <div className="space-y-2">
                  <label className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Notes or Summary</label>
                  <textarea
                    rows={6}
                    placeholder="Paste a summary, article, or notes to analyze..."
                    className="w-full bg-[#0a0a0b] border border-gray-800 rounded-xl px-4 py-3 focus:outline-none focus:border-indigo-500 transition-colors text-gray-200 resize-none"
                    value={formData.content}
                    onChange={(e) => setFormData({...formData, content: e.target.value})}
                  />
                </div>
              )}

              <button
                onClick={generatePath}
                disabled={loading}
                className="w-full bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 text-white font-semibold py-4 rounded-xl shadow-lg transition-all flex items-center justify-center gap-2 disabled:opacity-50"
              >
                {loading ? (
                  <Loader2 className="w-5 h-5 animate-spin" />
                ) : (
                  <>
                    <Sparkles className="w-5 h-5" />
                    Generate Learning Path
                  </>
                )}
              </button>
            </div>
          </div>
        )}

        {/* Path View */}
        {path && (
          <div className="space-y-8 animate-in slide-in-from-bottom-4 duration-700">
            <div className="flex items-center justify-between">
              <button 
                onClick={() => setPath(null)}
                className="text-sm text-gray-500 hover:text-white flex items-center gap-1 transition-colors"
              >
                <ArrowRight className="w-4 h-4 rotate-180" />
                Back to Input
              </button>
              <div className="px-3 py-1 bg-indigo-500/10 rounded-full border border-indigo-500/20 text-indigo-400 text-xs font-medium flex items-center gap-2">
                <Target className="w-3 h-3" />
                Personalized for You
              </div>
            </div>

            <div className="bg-[#16161a] p-8 rounded-3xl border border-gray-800">
              <h2 className="text-3xl font-bold mb-2">{path.topic}</h2>
              <p className="text-gray-400 mb-8 leading-relaxed">{path.description}</p>

              <div className="space-y-12 relative">
                {/* Vertical Line */}
                <div className="absolute left-[15px] top-4 bottom-4 w-0.5 bg-gradient-to-b from-indigo-500/50 via-gray-800 to-transparent" />

                {path.modules.map((module, mIdx) => (
                  <div key={mIdx} className="relative pl-12 group">
                    {/* Module Marker */}
                    <div className="absolute left-0 top-0 w-8 h-8 rounded-full bg-[#16161a] border-2 border-indigo-500 flex items-center justify-center shadow-[0_0_15px_rgba(99,102,241,0.3)] z-10 transition-transform group-hover:scale-110">
                      <BookOpen className="w-4 h-4 text-indigo-400" />
                    </div>

                    <div className="space-y-4">
                      <div>
                        <h3 className="text-xl font-bold text-white group-hover:text-indigo-400 transition-colors">
                          {module.title}
                        </h3>
                        {module.prerequisites?.length > 0 && (
                          <p className="text-xs text-amber-500/80 font-medium mt-1">
                            PREREQUISITES: {module.prerequisites.join(', ')}
                          </p>
                        )}
                      </div>

                      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                        {module.concepts.map((concept, cIdx) => (
                          <div 
                            key={cIdx}
                            className="bg-[#0a0a0b] p-4 rounded-xl border border-gray-800 hover:border-gray-700 transition-all flex items-center gap-3"
                          >
                            <Circle className="w-4 h-4 text-gray-600" />
                            <span className="text-sm text-gray-300">{concept}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                ))}
              </div>

              <div className="mt-12 pt-8 border-t border-gray-800 flex justify-center">
                <button className="flex items-center gap-2 px-8 py-3 bg-gray-100 text-black font-bold rounded-xl hover:bg-white transition-colors">
                  <CheckCircle2 className="w-5 h-5" />
                  Anchor to Knowledge Graph
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default LearningView;
