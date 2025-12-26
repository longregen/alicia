import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { api, PublicConfig } from '../services/api';

interface ConfigContextType {
  config: PublicConfig | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

const ConfigContext = createContext<ConfigContextType | undefined>(undefined);

export function ConfigProvider({ children }: { children: ReactNode }) {
  const [config, setConfig] = useState<PublicConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await api.getConfig();
      setConfig(data);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch configuration';
      setError(errorMessage);
      console.error('Failed to fetch config:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchConfig();
  }, []);

  return (
    <ConfigContext.Provider value={{ config, loading, error, refetch: fetchConfig }}>
      {children}
    </ConfigContext.Provider>
  );
}

export function useConfig(): ConfigContextType {
  const context = useContext(ConfigContext);
  if (context === undefined) {
    throw new Error('useConfig must be used within a ConfigProvider');
  }
  return context;
}
