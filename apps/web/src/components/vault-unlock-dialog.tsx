import { useState, type FormEvent } from "react";
import { type MeResponse } from "@/lib/api";
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

interface VaultUnlockDialogProps {
  user: MeResponse;
  open: boolean;
  onUnlock: (vaultKey: Uint8Array) => void;
  onLogout: () => Promise<void>;
}

export function VaultUnlockDialog({
  user,
  open,
  onUnlock,
  onLogout,
}: VaultUnlockDialogProps) {
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
        user.encrypted_vault_key,
        user.vault_key_nonce,
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
