import React, { createContext, useContext, useState, useEffect, useRef, useCallback } from 'react';
import { Brain, Loader2 } from 'lucide-react';

const WisdomContext = createContext();

// Google OAuth 2.0 Configuration
const GOOGLE_CLIENT_ID = "384412501694-q5h6p4r8764ilng1i2l4s6q8m2lic3e0.apps.googleusercontent.com";
const AUTH_SCOPE = "openid email profile";

export const WisdomProvider = ({ children }) => {
  const [view, setView] = useState('GRAPH');
  const [rigor, setRigor] = useState('LOW');
  const [activeNamespace, setActiveNamespace] = useState('ns-engineering');
  const [namespaces, setNamespaces] = useState([]);
  const [isInitializing, setIsInitializing] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [lastEvent, setLastEvent] = useState(null);
  const [user, setUser] = useState(null);

  const API_BASE = import.meta.env.VITE_ENGINE_URL || (window.location.hostname === 'localhost' ? 'http://localhost:8080' : '');
  const AGENT_BASE = import.meta.env.VITE_AGENT_URL || (window.location.hostname === 'localhost' ? 'http://localhost:8081' : '');
  const WS_BASE = import.meta.env.VITE_ENGINE_WS_URL || (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host;
  const AGENT_WS = import.meta.env.VITE_AGENT_WS_URL || (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host;
  
  const socketRef = useRef(null);

  const redirectToLogin = useCallback(() => {
    const redirectUri = import.meta.env.VITE_REDIRECT_URI || window.location.origin;
    const authUrl = `https://accounts.google.com/o/oauth2/v2/auth?client_id=${GOOGLE_CLIENT_ID}&redirect_uri=${encodeURIComponent(redirectUri)}&response_type=token&scope=${encodeURIComponent(AUTH_SCOPE)}&prompt=consent`;
    window.location.href = authUrl;
  }, []);

  const initRef = useRef(false);

  const fetchUser = useCallback(async (token) => {
    if (!token) return false;
    try {
      const headers = { 'Authorization': `Bearer ${token}` };
      const resp = await fetch(`${API_BASE}/whoami`, { headers });
      
      if (resp.ok) {
        const data = await resp.json();
        const enrichedUser = {
            ...data,
            is_admin: data.ldap === 'jesuscolin' || data.is_admin,
            role: data.ldap === 'jesuscolin' ? 'ADMIN' : (data.role || 'USER'),
            token: token
        };
        setUser(enrichedUser);
        setError(null);
        return true;
      } else if (resp.status === 401) {
        console.warn("Unauthorized, clearing token");
        localStorage.removeItem('wisdom_token');
        return false;
      } else {
        const errorText = await resp.text();
        setError(`Identity verification failed: ${resp.status} ${errorText}`);
        return false;
      }
    } catch (err) {
      console.error("Failed to fetch user:", err);
      setError("Cortex unreachable. Please check your connection.");
      return false;
    }
  }, [API_BASE]);

  // Effect 1: Authentication & Token Management
  useEffect(() => {
    if (initRef.current) return;
    initRef.current = true;

    const initializeAuth = async () => {
      setIsInitializing(true);
      try {
        const hash = window.location.hash;
        const params = new URLSearchParams(hash.substring(1));
        let token = params.get('access_token');

        if (token) {
          localStorage.setItem('wisdom_token', token);
          window.history.replaceState(null, null, window.location.pathname);
        } else {
          token = localStorage.getItem('wisdom_token');
        }

        if (token) {
          const success = await fetchUser(token);
          if (!success) {
             redirectToLogin();
          } else {
             setIsInitializing(false);
          }
        } else {
          redirectToLogin();
        }
      } catch (e) {
        console.error("Auth init error:", e);
        setError("System initialization failed.");
        setIsInitializing(false);
      }
    };

    initializeAuth();
  }, [fetchUser, redirectToLogin]);

  // Effect 2: Shared WebSocket Connection
  useEffect(() => {
    if (!user) return;

    const socket = new WebSocket(`${WS_BASE}/ws`);
    
    socket.onopen = () => {
      console.log("WebSocket connected to:", WS_BASE);
      setError(null);
    };

    socket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        setLastEvent(data);
      } catch (err) {
        console.error("Failed to parse WS message:", err);
      }
    };

    socket.onerror = (event) => {
      console.error("WebSocket error:", event);
      console.warn("WebSocket connection failed");
    };

    socket.onclose = () => {
      console.log("WebSocket connection closed");
    };

    socketRef.current = socket;

    return () => {
      if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
        socket.close();
      }
    };
  }, [WS_BASE, user]);

  const logout = () => {
    localStorage.removeItem('wisdom_token');
    setUser(null);
    window.location.reload();
  };

  const sendWS = useCallback((type, payload) => {
    if (socketRef.current && socketRef.current.readyState === WebSocket.OPEN) {
      socketRef.current.send(JSON.stringify({ type, payload }));
      return true;
    }
    return false;
  }, []);

  if (isInitializing) {
    return (
      <div className="flex flex-col items-center justify-center h-screen bg-[#0d1117] text-white overflow-hidden relative">
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-indigo-500/10 rounded-full blur-[120px] animate-pulse" />
        <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-blue-500/10 rounded-full blur-[120px] animate-pulse delay-700" />
        
        <div className="relative z-10 flex flex-col items-center gap-6">
          <div className="p-5 bg-indigo-500/10 rounded-3xl border border-indigo-500/20 shadow-[0_0_50px_rgba(99,102,241,0.15)] animate-bounce duration-[2000ms]">
            <Brain className="text-indigo-400 w-16 h-16" />
          </div>
          
          <div className="flex flex-col items-center gap-2">
            <h2 className="text-2xl font-black tracking-[0.2em] uppercase italic text-indigo-100">Wisdom</h2>
            <div className="flex items-center gap-3 px-4 py-2 bg-white/5 rounded-full border border-white/10 backdrop-blur-md">
              <Loader2 className="text-indigo-400 animate-spin w-4 h-4" />
              <span className="text-[10px] font-bold text-indigo-300 tracking-[0.3em] uppercase">Initializing Neural Atlas</span>
            </div>
          </div>
        </div>

        {error && (
          <div className="absolute bottom-12 px-6 py-3 bg-red-500/10 border border-red-500/20 rounded-2xl flex items-center gap-3 backdrop-blur-xl">
            <div className="w-2 h-2 rounded-full bg-red-500 animate-pulse" />
            <span className="text-sm font-bold text-red-200">{error}</span>
          </div>
        )}
      </div>
    );
  }

  return (
    <WisdomContext.Provider value={{
      view, setView,
      rigor, setRigor,
      activeNamespace, setActiveNamespace,
      namespaces, setNamespaces,
      user, setUser,
      loading, setLoading,
      error, setError,
      API_BASE,
      AGENT_BASE,
      WS_BASE,
      AGENT_WS,
      lastEvent,
      socketRef,
      sendWS,
      logout
    }}>
      {children}
    </WisdomContext.Provider>
  );
};

export const useWisdom = () => useContext(WisdomContext);
