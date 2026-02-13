import { useEffect, useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth";
import { api, getApiError, type GetPasteResponse } from "@/lib/api";
import { decryptPaste, decryptPasteWithKey } from "@/lib/crypto";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { DateRange } from "@/components/dashboard/date-range";
import { PageSkeleton } from "../../components/ui/page-skeleton";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";

export const Route = createFileRoute("/pastes/$id")({
  component: ViewPaste,
});

type Status = "loading" | "ready" | "not_found" | "missing_key" | "decrypt_error" | "error";

const ERROR_STATES: Record<
  Exclude<Status, "loading" | "ready">,
  { title: string; description: string }
> = {
  not_found: {
    title: "paste not found",
    description:
      "This paste does not exist, has expired, or was burned after reading.",
  },
  missing_key: {
    title: "missing decryption key",
    description:
      "The share link must include the decryption key in the URL fragment (#key=...). Log in to decrypt pastes you own.",
  },
  decrypt_error: {
    title: "decryption failed",
    description:
      "The decryption key is invalid or the paste data is corrupted.",
  },
  error: {
    title: "error",
    description: "Something went wrong while loading this paste.",
  },
};

function ErrorState({ status }: { status: Exclude<Status, "loading" | "ready"> }) {
  const { title, description } = ERROR_STATES[status];
  return (
    <div className="mx-auto max-w-2xl px-4 py-10 space-y-6">
      <div className="space-y-1">
        <p className="text-sm font-medium">
          <span className="text-muted-foreground">$ </span>
          {title}
        </p>
        <p className="text-xs text-muted-foreground">{description}</p>
      </div>
    </div>
  );
}

function getKeyFromHash(): string | null {
  const hash = window.location.hash;
  if (!hash || hash === "#") return null;
  const raw = hash.slice(1);
  const params = new URLSearchParams(raw.startsWith("?") ? raw.slice(1) : raw);
  return params.get("key");
}

const RELATIVE_TIME_FORMATTER = new Intl.RelativeTimeFormat(undefined, {
  numeric: "auto",
});

function formatRelativeExpiration(dateString: string): string {
  const expiresAt = new Date(dateString);
  if (Number.isNaN(expiresAt.getTime())) return "unknown";
  if (expiresAt.getFullYear() >= 2100) return "never";

  const deltaSeconds = Math.round((expiresAt.getTime() - Date.now()) / 1000);
  const absSeconds = Math.abs(deltaSeconds);

  if (absSeconds < 60) {
    return RELATIVE_TIME_FORMATTER.format(deltaSeconds, "second");
  }
  if (absSeconds < 3600) {
    return RELATIVE_TIME_FORMATTER.format(Math.round(deltaSeconds / 60), "minute");
  }
  if (absSeconds < 86_400) {
    return RELATIVE_TIME_FORMATTER.format(Math.round(deltaSeconds / 3600), "hour");
  }
  if (absSeconds < 2_592_000) {
    return RELATIVE_TIME_FORMATTER.format(Math.round(deltaSeconds / 86_400), "day");
  }

  return RELATIVE_TIME_FORMATTER.format(Math.round(deltaSeconds / 2_592_000), "month");
}

function ViewPaste() {
  const { isLoading } = useAuth();

  if (isLoading) {
    return <PageSkeleton />;
  }

  return <PasteContent />;
}

function PasteContent() {
  const { id } = Route.useParams();
  const navigate = useNavigate();
  const { vaultKey } = useAuth();
  const isLoggedIn = !!vaultKey;

  const [status, setStatus] = useState<Status>("loading");
  const [paste, setPaste] = useState<GetPasteResponse | null>(null);
  const [decrypted, setDecrypted] = useState<{
    title: string;
    body: string;
  } | null>(null);
  const [hashKey, setHashKey] = useState<string | null>(() => getKeyFromHash());
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  useEffect(() => {
    const update = () => setHashKey(getKeyFromHash());
    window.addEventListener("hashchange", update);
    return () => window.removeEventListener("hashchange", update);
  }, []);

  useEffect(() => {
    setStatus("loading");
    setPaste(null);
    setDecrypted(null);

    api
      .getPaste(id)
      .then((data) => {
        setPaste(data);
      })
      .catch((err) => {
        const apiErr = getApiError(err);
        if (apiErr?.code === "NOT_FOUND") {
          setStatus("not_found");
        } else {
          setStatus("error");
        }
      });
  }, [id]);

  useEffect(() => {
    if (!paste) return;

    try {
      if (hashKey) {
        const result = decryptPasteWithKey(
          hashKey,
          paste.title_ciphertext,
          paste.title_nonce,
          paste.body_ciphertext,
          paste.body_nonce,
        );
        setDecrypted(result);
        setStatus("ready");
        return;
      }

      if (vaultKey) {
        const result = decryptPaste(
          vaultKey,
          paste.encrypted_paste_key_ciphertext,
          paste.encrypted_paste_key_nonce,
          paste.title_ciphertext,
          paste.title_nonce,
          paste.body_ciphertext,
          paste.body_nonce,
        );
        setDecrypted(result);
        setStatus("ready");
        return;
      }

      setStatus("missing_key");
    } catch {
      setStatus("decrypt_error");
    }
  }, [paste, hashKey, vaultKey]);

  if (status === "loading") {
    return <PageSkeleton />;
  }

  if (status !== "ready") {
    return <ErrorState status={status} />;
  }

  const isOwnerView = hashKey === null;
  const relativeExpiry = formatRelativeExpiration(paste!.expires_at);
  const expiresSoon = (() => {
    const expiresAt = new Date(paste!.expires_at).getTime();
    if (Number.isNaN(expiresAt)) return false;
    return expiresAt > Date.now() && expiresAt - Date.now() < 3_600_000;
  })();

  const handleDelete = async () => {
    setDeleteError(null);
    setIsDeleting(true);
    try {
      await api.deletePaste(id);
      setDeleteOpen(false);
      navigate({ to: "/dashboard", search: { tab: "pastes" } });
    } catch {
      setDeleteError("Failed to delete paste.");
      setIsDeleting(false);
    }
  };

  return (
    <div className="mx-auto max-w-2xl px-4 py-10 space-y-6">
      {isLoggedIn && (
        <Link
          to="/dashboard"
          search={{ tab: "pastes" }}
          className="text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          ← dashboard
        </Link>
      )}
      <div className="space-y-1">
        <p className="text-sm font-medium">
          <span className="text-muted-foreground">$ </span>
          view paste
        </p>
        <p className="text-xs text-muted-foreground">
          Decrypted client-side.
        </p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-start justify-between gap-3">
            <div className="space-y-2 min-w-0">
              <CardTitle className="break-words">{decrypted!.title}</CardTitle>
              <div className="flex flex-wrap items-center gap-2">
                {paste!.burn_after_read && (
                  <Badge variant="outline" className="text-destructive">
                    burned
                  </Badge>
                )}
                <Badge variant="outline" className="text-muted-foreground">
                  expires {relativeExpiry}
                </Badge>
                {expiresSoon && (
                  <Badge variant="outline" className="text-amber-600">
                    expiring soon
                  </Badge>
                )}
                <Badge variant="outline" className="text-muted-foreground">
                  {hashKey ? "shared key" : "vault key"}
                </Badge>
              </div>
            </div>
            {isOwnerView && (
              <AlertDialog
                open={deleteOpen}
                onOpenChange={(open) => {
                  setDeleteOpen(open);
                  if (!open) setDeleteError(null);
                }}
              >
                <AlertDialogTrigger render={<Button variant="destructive" size="xs" />}>
                  Delete
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Delete paste?</AlertDialogTitle>
                    <AlertDialogDescription>
                      This will permanently delete this paste. This action cannot
                      be undone.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter className="flex-col items-stretch gap-3">
                    {deleteError && (
                      <p className="text-xs text-destructive order-last w-full">
                        {deleteError}
                      </p>
                    )}
                    <AlertDialogCancel disabled={isDeleting}>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={handleDelete}
                      disabled={isDeleting}
                    >
                      {isDeleting ? "Deleting..." : "Delete paste"}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            )}
          </div>
          <CardDescription>
            <DateRange created={paste!.created_at} expires={paste!.expires_at} />
          </CardDescription>
        </CardHeader>

        <Separator />

        <CardContent className="pt-4">
          <pre className="whitespace-pre-wrap break-words font-mono text-sm">
            {decrypted!.body}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
}
