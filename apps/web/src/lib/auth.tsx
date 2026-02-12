import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import { api, type MeResponse } from "@/lib/api";
import { deriveLoginKeys, decryptVaultKey } from "@/lib/crypto";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogAction,
} from "@/components/ui/alert-dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Field, FieldLabel, FieldError } from "@/components/ui/field";

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

function VaultUnlockDialog({
  user,
  open,
  onUnlock,
  onLogout,
}: {
  user: MeResponse;
  open: boolean;
  onUnlock: (vaultKey: Uint8Array) => void;
  onLogout: () => Promise<void>;
}) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    try {
      const { masterKey } = await deriveLoginKeys(password, user.salt);
      const vaultKey = await decryptVaultKey(
        masterKey,
        user.encryptedVaultKey,
        user.vaultKeyNonce,
      );
      onUnlock(vaultKey);
    } catch {
      setError("Wrong password. Please try again.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AlertDialog open={open}>
      <AlertDialogContent>
        <form onSubmit={handleSubmit}>
          <AlertDialogHeader>
            <AlertDialogTitle>Unlock vault</AlertDialogTitle>
            <AlertDialogDescription>
              Enter your password to decrypt your vault.
            </AlertDialogDescription>
          </AlertDialogHeader>

          <Field data-invalid={!!error || undefined}>
            <FieldLabel htmlFor="vault-password">Password</FieldLabel>
            <Input
              id="vault-password"
              type="password"
              autoFocus
              value={password}
              onChange={(e) => {
                setPassword(e.target.value);
                setError(null);
              }}
              disabled={submitting}
              required
            />
            {error && <FieldError>{error}</FieldError>}
          </Field>

          <AlertDialogFooter className="mt-4">
            <Button
              type="button"
              variant="link"
              size="sm"
              disabled={submitting}
              onClick={onLogout}
            >
              Logout
            </Button>
            <AlertDialogAction type="submit" disabled={submitting}>
              {submitting ? "Unlocking…" : "Unlock"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </form>
      </AlertDialogContent>
    </AlertDialog>
  );
}

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

  const showUnlockDialog = user !== null && vaultKey === null && !isLoading;

  return (
    <AuthContext.Provider
      value={{ user, vaultKey, isLoading, setVaultKey, clearVaultKey, logout, refetchUser }}
    >
      {children}
      {user && (
        <VaultUnlockDialog
          user={user}
          open={showUnlockDialog}
          onUnlock={setVaultKey}
          onLogout={logout}
        />
      )}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
