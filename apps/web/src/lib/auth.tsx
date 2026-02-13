import {
  createContext,
  lazy,
  Suspense,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { api, type MeResponse } from "@/lib/api";

const VaultUnlockDialog = lazy(
  () =>
    import("@/components/vault-unlock-dialog").then((m) => ({
      default: m.VaultUnlockDialog,
    })),
);

interface AuthState {
  user: MeResponse | null;
  vaultKey: Uint8Array | null;
  isLoading: boolean;
}

interface AuthContextValue extends AuthState {
  setVaultKey: (key: Uint8Array) => void;
  clearVaultKey: () => void;
  logout: () => Promise<void>;
  refetchUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<MeResponse | null>(null);
  const [vaultKey, setVaultKeyState] = useState<Uint8Array | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetchUser = useCallback(async () => {
    try {
      const me = await api.me();
      setUser(me);
    } catch {
      setUser(null);
    }
  }, []);

  useEffect(() => {
    fetchUser().finally(() => setIsLoading(false));
  }, [fetchUser]);

  const setVaultKey = useCallback((key: Uint8Array) => {
    setVaultKeyState(key);
  }, []);

  const clearVaultKey = useCallback(() => {
    setVaultKeyState(null);
  }, []);

  const logout = useCallback(async () => {
    await api.logout();
    setUser(null);
    setVaultKeyState(null);
  }, []);

  const refetchUser = useCallback(async () => {
    await fetchUser();
  }, [fetchUser]);

  const [suppressUnlock, setSuppressUnlock] = useState(false);

  useEffect(() => {
    const check = () => {
      const onPasteRoute = window.location.pathname.startsWith("/pastes/");
      const hash = window.location.hash;
      const raw = hash.startsWith("#") ? hash.slice(1) : hash;
      const params = new URLSearchParams(raw.startsWith("?") ? raw.slice(1) : raw);
      setSuppressUnlock(onPasteRoute && !!params.get("key"));
    };
    check();
    window.addEventListener("hashchange", check);
    window.addEventListener("popstate", check);
    return () => {
      window.removeEventListener("hashchange", check);
      window.removeEventListener("popstate", check);
    };
  }, []);

  const showUnlockDialog = user !== null && vaultKey === null && !isLoading && !suppressUnlock;

  return (
    <AuthContext.Provider
      value={{ user, vaultKey, isLoading, setVaultKey, clearVaultKey, logout, refetchUser }}
    >
      {children}
      {user && (
        <Suspense fallback={null}>
          <VaultUnlockDialog
            user={user}
            open={showUnlockDialog}
            onUnlock={setVaultKey}
            onLogout={logout}
          />
        </Suspense>
      )}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
