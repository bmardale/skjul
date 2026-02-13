import { useCallback, useState } from "react";
import { Link } from "@tanstack/react-router";
import { HugeiconsIcon } from "@hugeicons/react";
import { Cancel01Icon, Attachment01Icon } from "@hugeicons/core-free-icons";
import { api, type PasteListItem } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { decryptPasteTitle } from "@/lib/crypto";
import { useAsyncData } from "@/lib/hooks/use-async-data";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { DataCard } from "@/components/dashboard/data-card";
import { DateRange } from "@/components/dashboard/date-range";

interface DecryptedPaste {
  id: string;
  title: string;
  burnAfterRead: boolean;
  createdAt: string;
  expiresAt: string;
  decryptError: boolean;
  attachmentCount: number;
}

function decryptPasteListItem(
  vaultKey: Uint8Array,
  paste: PasteListItem,
): DecryptedPaste {
  try {
    const title = decryptPasteTitle(
      vaultKey,
      paste.encrypted_paste_key_ciphertext,
      paste.encrypted_paste_key_nonce,
      paste.title_ciphertext,
      paste.title_nonce,
    );
    return {
      id: paste.id,
      title,
      burnAfterRead: paste.burn_after_read,
      createdAt: paste.created_at,
      expiresAt: paste.expires_at,
      decryptError: false,
      attachmentCount: paste.attachment_count ?? 0,
    };
  } catch {
    return {
      id: paste.id,
      title: "Unable to decrypt",
      burnAfterRead: paste.burn_after_read,
      createdAt: paste.created_at,
      expiresAt: paste.expires_at,
      decryptError: true,
      attachmentCount: paste.attachment_count ?? 0,
    };
  }
}

export function PastesCard({ isActive }: { isActive: boolean }) {
  const { vaultKey } = useAuth();
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);
  const fetchPastes = useCallback(async () => {
    if (!vaultKey) return [];
    const data = await api.listPastes();
    return data.map((paste) => decryptPasteListItem(vaultKey, paste));
  }, [vaultKey]);

  const { data: pastes, setData: setPastes, loading, error, clearError, refresh } =
    useAsyncData<DecryptedPaste[]>(fetchPastes, {
      enabled: isActive && !!vaultKey,
      initialData: [],
      errorMessage: "Failed to load pastes.",
    });

  const handleDelete = async () => {
    const id = deleteTarget;
    if (!id) return;
    setLocalError(null);
    setIsDeleting(true);
    try {
      await api.deletePaste(id);
      setPastes((prev) => prev.filter((paste) => paste.id !== id));
      setDeleteTarget(null);
    } catch {
      setLocalError("Failed to delete paste.");
    } finally {
      setIsDeleting(false);
    }
  };

  const handleRefresh = async () => {
    setLocalError(null);
    clearError();
    await refresh();
  };

  const displayError = localError || error;

  return (
    <>
      <DataCard
        title="Pastes"
        description="Your encrypted pastes. Titles are decrypted client-side."
        loading={loading}
        error={displayError}
        empty={pastes.length === 0}
        emptyMessage="No pastes yet."
        onRefresh={() => void handleRefresh()}
        refreshLabel="Refresh pastes"
      >
        <div className="space-y-0">
          {pastes.map((paste, i) => (
            <div key={paste.id}>
              {i > 0 && <Separator className="my-3" />}
              <Link
                to="/pastes/$id"
                params={{ id: paste.id }}
                className="block rounded-md px-2 py-2 hover:bg-muted/30 transition-colors"
              >
                <div className="flex items-center justify-between gap-4">
                  <div className="min-w-0 space-y-1">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-medium font-mono truncate">
                        {paste.title}
                      </span>
                      {paste.burnAfterRead && (
                        <Badge variant="outline" className="text-destructive">
                          burn
                        </Badge>
                      )}
                      {paste.decryptError && (
                        <Badge variant="outline" className="text-muted-foreground">
                          encrypted
                        </Badge>
                      )}
                      {paste.attachmentCount > 0 && (
                        <Badge variant="outline" className="text-muted-foreground">
                          <HugeiconsIcon icon={Attachment01Icon} size={10} className="mr-1" />
                          {paste.attachmentCount}
                        </Badge>
                      )}
                    </div>
                    <DateRange
                      created={paste.createdAt}
                      expires={paste.expiresAt}
                    />
                  </div>
                  <Button
                    variant="destructive"
                    size="xs"
                    disabled={isDeleting}
                    aria-label="Delete paste"
                    title="Delete paste"
                    onClick={(e) => {
                      e.preventDefault();
                      setDeleteTarget(paste.id);
                    }}
                  >
                    <HugeiconsIcon icon={Cancel01Icon} size={12} />
                  </Button>
                </div>
              </Link>
            </div>
          ))}
        </div>
      </DataCard>

      <AlertDialog
        open={!!deleteTarget}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteTarget(null);
            setLocalError(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete paste?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete this paste. This action cannot be
              undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => void handleDelete()}
              disabled={isDeleting}
            >
              {isDeleting ? "Deleting..." : "Delete paste"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
