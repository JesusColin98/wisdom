import React, { createContext, useContext, useState, useEffect, useRef, useCallback } from 'react';

const WisdomContext = createContext();

// Google OAuth 2.0 Configuration
const GOOGLE_CLIENT_ID = "384412501694-q5h6p4r8764ilng1i2l4s6q8m2lic3e0.apps.googleusercontent.com";
const AUTH_SCOPE = "openid email profile";

export const WisdomProvider = ({ children }) => {
  const [view, setView] = useState('GRAPH');
  const [rigor, setRigor] = useState('LOW');
  const [activeNamespace, setActiveNamespace] = useState('ns-engineering');
  const [namespaces, setNamespaces] = useState([]);
  const [user, setUser] = useState(null); // Initialize as null to trigger login check
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [lastEvent, setLastEvent] = useState(null);

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

  const fetchUser = useCallback(async (token) => {
    try {
      setLoading(true);
      const headers = token ? { 'Authorization': `Bearer ${token}` } : {};
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
        setLoading(false);
        return true;
      } else if (resp.status === 401) {
        console.warn("Unauthorized, clearing token");
        localStorage.removeItem('wisdom_token');
        return false;
      }
    } catch (err) {
      console.error("Failed to fetch user:", err);
    }
    setLoading(false);
    return false;
  }, [API_BASE]);

  useEffect(() => {
    const initializeAuth = async () => {
      // 1. Check for token in URL (OAuth callback)
      const hash = window.location.hash;
      const params = new URLSearchParams(hash.substring(1));
      let token = params.get('access_token');

      if (token) {
        console.log("Captured token from URL");
        localStorage.setItem('wisdom_token', token);
        // Clear hash from URL
        window.history.replaceState(null, null, window.location.pathname);
      } else {
        // 2. Check for token in localStorage
        token = localStorage.getItem('wisdom_token');
      }

      if (token) {
        const success = await fetchUser(token);
        if (!success) {
          redirectToLogin();
        }
      } else {
        redirectToLogin();
      }
    };

    initializeAuth();

    // Shared WebSocket
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
      setError("WebSocket connection failed");
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
  }, [WS_BASE, fetchUser, redirectToLogin]);

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
    console.warn("WebSocket not open. ReadyState:", socketRef.current?.readyState);
    return false;
  }, []);

  if (loading) {
    return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', background: '#0a0a0a', color: '#fff' }}>Loading Wisdom...</div>;
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
