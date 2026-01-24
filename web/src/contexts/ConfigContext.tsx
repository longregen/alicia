import { createContext, useState, ReactNode } from 'react';

/**
 * Stub config context - the getConfig endpoint has been removed from the backend.
 * This provides a minimal implementation for components that expect a config context.
 */
interface PublicConfig {
  livekit_url?: string;
}

interface ConfigContextType {
  config: PublicConfig | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

const ConfigContext = createContext<ConfigContextType | undefined>(undefined);

export function ConfigProvider({ children }: { children: ReactNode }) {
  const liveKitUrl = import.meta.env.VITE_LIVEKIT_URL || undefined;

  const [config] = useState<PublicConfig | null>({ livekit_url: liveKitUrl });
  const loading = false;
  const error = null;

  const refetch = async () => {
    // No-op - config endpoint doesn't exist
    console.warn('Config endpoint is not available in the current backend');
  };

  return (
    <ConfigContext.Provider value={{ config, loading, error, refetch }}>
      {children}
    </ConfigContext.Provider>
  );
}
