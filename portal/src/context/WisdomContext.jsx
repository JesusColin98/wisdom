import React, { createContext, useContext, useState, useEffect } from 'react';

const WisdomContext = createContext();

export const WisdomProvider = ({ children }) => {
  const [view, setView] = useState('GRAPH');
  const [rigor, setRigor] = useState('LOW');
  const [activeNamespace, setActiveNamespace] = useState('ns-engineering');
  const [user, setUser] = useState({ ldap: 'anonymous', role: 'USER', is_admin: false });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const API_BASE = window.location.hostname === 'localhost' ? 'http://localhost:8080' : '';

  const fetchUser = async () => {
    try {
      const resp = await fetch(`${API_BASE}/whoami`);
      if (resp.ok) {
        const data = await resp.json();
        setUser(data);
      }
    } catch (err) {
      console.error("Failed to fetch user:", err);
    }
  };

  useEffect(() => {
    fetchUser();
  }, []);

  return (
    <WisdomContext.Provider value={{
      view, setView,
      rigor, setRigor,
      activeNamespace, setActiveNamespace,
      user, setUser,
      loading, setLoading,
      error, setError,
      API_BASE
    }}>
      {children}
    </WisdomContext.Provider>
  );
};

export const useWisdom = () => useContext(WisdomContext);
