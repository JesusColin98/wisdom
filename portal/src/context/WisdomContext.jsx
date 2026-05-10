import React, { createContext, useContext, useState, useEffect, useRef, useCallback } from 'react';

const WisdomContext = createContext();

export const WisdomProvider = ({ children }) => {
  const [view, setView] = useState('GRAPH');
  const [rigor, setRigor] = useState('LOW');
  const [activeNamespace, setActiveNamespace] = useState('ns-engineering');
  const [user, setUser] = useState({ ldap: 'anonymous', role: 'USER', is_admin: false });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [lastEvent, setLastEvent] = useState(null);

  const API_BASE = import.meta.env.VITE_ENGINE_URL || (window.location.hostname === 'localhost' ? 'http://localhost:8080' : '');
  const AGENT_BASE = import.meta.env.VITE_AGENT_URL || (window.location.hostname === 'localhost' ? 'http://localhost:8081' : '');
  const WS_BASE = import.meta.env.VITE_ENGINE_WS_URL || (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host;
  const AGENT_WS = import.meta.env.VITE_AGENT_WS_URL || (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host;
  
  const socketRef = useRef(null);

  const fetchUser = useCallback(async () => {
    try {
      const resp = await fetch(`${API_BASE}/whoami`);
      if (resp.ok) {
        const data = await resp.json();
        // Grant admin status if LDAP is jesuscolin
        const enrichedUser = {
            ...data,
            is_admin: data.ldap === 'jesuscolin' || data.is_admin,
            role: data.ldap === 'jesuscolin' ? 'ADMIN' : (data.role || 'USER')
        };
        setUser(enrichedUser);
      }
    } catch (err) {
      console.error("Failed to fetch user:", err);
    }
  }, [API_BASE]);

  useEffect(() => {
    const timer = setTimeout(() => {
      fetchUser();
    }, 0);

    // Shared WebSocket
    const socket = new WebSocket(`${WS_BASE}/ws`);
    socket.onmessage = (event) => {
      const data = JSON.parse(event.data);
      setLastEvent(data);
    };
    socketRef.current = socket;

    return () => {
      clearTimeout(timer);
      socket.close();
    };
  }, [WS_BASE, fetchUser]);

  return (
    <WisdomContext.Provider value={{
      view, setView,
      rigor, setRigor,
      activeNamespace, setActiveNamespace,
      user, setUser,
      loading, setLoading,
      error, setError,
      API_BASE,
      AGENT_BASE,
      WS_BASE,
      AGENT_WS,
      lastEvent,
      socketRef
    }}>
      {children}
    </WisdomContext.Provider>
  );
};

export const useWisdom = () => useContext(WisdomContext);
