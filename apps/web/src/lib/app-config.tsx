import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { api, type PublicConfig } from "@/lib/api";

interface AppConfigContextValue {
  config: PublicConfig | null;
  isLoading: boolean;
  refetch: () => Promise<void>;
}

const AppConfigContext = createContext<AppConfigContextValue | null>(null);

export function AppConfigProvider({ children }: { children: ReactNode }) {
  const [config, setConfig] = useState<PublicConfig | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetchConfig = useCallback(async () => {
    try {
      const data = await api.getPublicConfig();
      setConfig(data);
    } catch {
      setConfig({ require_invite_code: false });
    }
  }, []);

  useEffect(() => {
    fetchConfig().finally(() => setIsLoading(false));
  }, [fetchConfig]);

  const refetch = useCallback(async () => {
    await fetchConfig();
  }, [fetchConfig]);

  return (
    <AppConfigContext.Provider value={{ config, isLoading, refetch }}>
      {children}
    </AppConfigContext.Provider>
  );
}

export function useAppConfig(): AppConfigContextValue {
  const ctx = useContext(AppConfigContext);
  if (!ctx) throw new Error("useAppConfig must be used within AppConfigProvider");
  return ctx;
}
