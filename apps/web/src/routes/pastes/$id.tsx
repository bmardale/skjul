import { useEffect, useState, useRef } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth";
import {
  api,
  getApiError,
  type GetPasteResponse,
  type GetPasteMetaResponse,
  type PasteAttachment,
} from "@/lib/api";
import {
  decryptPaste,
  decryptPasteWithKey,
  decryptFile,
  decryptFilename,
  getPasteKeyFromHash,
  getPasteKeyFromVault,
  derivePasteSubkeys,
  AAD,
} from "@/lib/crypto";
import { hexToBytes } from "@noble/hashes/utils.js";
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
import { PasteBody } from "@/components/paste-body";

export const Route = createFileRoute("/pastes/$id")({
  component: ViewPaste,
});

type Status =
  | "loading"
  | "confirm_required"
  | "ready"
  | "not_found"
  | "missing_key"
  | "decrypt_error"
  | "rate_limited"
  | "error";

const ERROR_STATES: Record<
  Exclude<Status, "loading" | "ready" | "confirm_required">,
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
  rate_limited: {
    title: "too many requests",
    description: "Please wait a moment and try again.",
  },
  error: {
    title: "error",
    description: "Something went wrong while loading this paste.",
  },
};

function ErrorState({
  status,
}: {
  status: Exclude<Status, "loading" | "ready" | "confirm_required">;
}) {
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

type ImageLoadState = "idle" | "loading" | "loaded" | "error";

function AttachmentItem({
  att,
  pasteKey,
  onDownload,
  downloading,
}: {
  att: PasteAttachment;
  pasteKey: Uint8Array;
  onDownload: (att: PasteAttachment) => void;
  downloading: boolean;
}) {
  const [imageState, setImageState] = useState<ImageLoadState>("idle");
  const [objectUrl, setObjectUrl] = useState<string | null>(null);
  const [lightboxOpen, setLightboxOpen] = useState(false);
  const containerRef = useRef<HTMLLIElement>(null);
  const [inView, setInView] = useState(false);

  const { metaKey, fileKey } = derivePasteSubkeys(pasteKey);

  const getMimeType = (): string | null => {
    if (!att.mime_ciphertext || !att.mime_nonce || pasteKey.length === 0) return null;
    try {
      return decryptFilename(att.mime_ciphertext, att.mime_nonce, metaKey, AAD.MIME);
    } catch {
      return null;
    }
  };

  const getFilename = () => {
    try {
      return decryptFilename(
        att.filename_ciphertext,
        att.filename_nonce,
        metaKey,
        AAD.FILENAME
      );
    } catch {
      return "encrypted file";
    }
  };

  const mimeType = getMimeType();
  const isImage = mimeType?.startsWith("image/") ?? false;

  useEffect(() => {
    if (!inView || !isImage || imageState !== "idle" || pasteKey.length === 0) return;
    setImageState("loading");
    fetch(att.download_url)
      .then((res) => {
        if (!res.ok) throw new Error("Fetch failed");
        return res.arrayBuffer();
      })
      .then((buf) => {
        const ciphertext = new Uint8Array(buf);
        const plaintext = decryptFile(
          ciphertext,
          hexToBytes(att.content_nonce),
          fileKey,
          AAD.FILE
        );
        const copy = new Uint8Array(plaintext.length);
        copy.set(plaintext);
        const blob = new Blob([copy], { type: mimeType ?? undefined });
        setObjectUrl(URL.createObjectURL(blob));
        setImageState("loaded");
      })
      .catch(() => setImageState("error"));
  }, [att, pasteKey, inView, isImage, imageState, mimeType]);

  useEffect(() => {
    const el = containerRef.current;
    if (!el || !isImage) return;
    const obs = new IntersectionObserver(
      ([entry]) => {
        if (entry?.isIntersecting) setInView(true);
      },
      { rootMargin: "100px" }
    );
    obs.observe(el);
    return () => obs.disconnect();
  }, [isImage]);

  useEffect(() => () => {
    if (objectUrl) URL.revokeObjectURL(objectUrl);
  }, [objectUrl]);

  const filename = getFilename();

  return (
    <li
      ref={containerRef}
      className="rounded-md border bg-muted/30 overflow-hidden"
    >
      {isImage && (
        <div className="p-2 space-y-2">
          {imageState === "idle" && (
            <div className="aspect-video rounded bg-muted animate-pulse" />
          )}
          {imageState === "loading" && (
            <div className="aspect-video rounded bg-muted animate-pulse flex items-center justify-center text-xs text-muted-foreground">
              Loading…
            </div>
          )}
          {imageState === "loaded" && objectUrl && (
            <button
              type="button"
              className="block w-full max-w-sm rounded overflow-hidden hover:opacity-90 transition-opacity"
              onClick={() => setLightboxOpen(true)}
            >
              <img
                src={objectUrl}
                alt={filename}
                className="max-h-48 w-auto object-contain"
              />
            </button>
          )}
          {imageState === "error" && (
            <div className="aspect-video rounded bg-muted flex items-center justify-center text-xs text-muted-foreground">
              Failed to load preview
            </div>
          )}
          {lightboxOpen && objectUrl && (
            <div
              className="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4"
              onClick={() => setLightboxOpen(false)}
            >
              <img
                src={objectUrl}
                alt={filename}
                className="max-w-full max-h-full object-contain"
                onClick={(e) => e.stopPropagation()}
              />
            </div>
          )}
        </div>
      )}
      <div className="flex items-center justify-between gap-2 px-3 py-2 sm:flex-row flex-wrap">
        <span className="truncate text-sm" title={filename}>
          {filename}
        </span>
        <span className="shrink-0 text-muted-foreground text-xs">
          {(att.encrypted_size / 1024).toFixed(1)} KB
        </span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={downloading}
          onClick={() => onDownload(att)}
        >
          {downloading ? "Downloading..." : "Download"}
        </Button>
      </div>
    </li>
  );
}

function AttachmentList({
  attachments,
  pasteKey,
}: {
  attachments: PasteAttachment[];
  pasteKey: Uint8Array;
}) {
  const [downloading, setDownloading] = useState<string | null>(null);

  const { fileKey, metaKey } = derivePasteSubkeys(pasteKey);

  const handleDownload = async (att: PasteAttachment) => {
    if (pasteKey.length === 0) return;
    setDownloading(att.id);
    try {
      const res = await fetch(att.download_url);
      if (!res.ok) throw new Error("Download failed");
      const ciphertext = new Uint8Array(await res.arrayBuffer());
      const plaintext = decryptFile(
        ciphertext,
        hexToBytes(att.content_nonce),
        fileKey,
        AAD.FILE
      );
      const filename = decryptFilename(
        att.filename_ciphertext,
        att.filename_nonce,
        metaKey,
        AAD.FILENAME
      );
      const copy = new Uint8Array(plaintext.length);
      copy.set(plaintext);
      const blob = new Blob([copy]);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      a.click();
      URL.revokeObjectURL(url);
    } finally {
      setDownloading(null);
    }
  };

  return (
    <div className="space-y-2">
      <p className="text-xs font-medium text-muted-foreground">Attachments</p>
      <ul className="space-y-2">
        {attachments.map((att) => (
          <AttachmentItem
            key={att.id}
            att={att}
            pasteKey={pasteKey}
            onDownload={handleDownload}
            downloading={downloading === att.id}
          />
        ))}
      </ul>
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

function BurnConfirmInterstitial({
  meta,
  isLoggedIn,
  isRevealing,
  revealError,
  onReveal,
  onBack,
}: {
  meta: GetPasteMetaResponse;
  isLoggedIn: boolean;
  isRevealing: boolean;
  revealError: string | null;
  onReveal: () => void;
  onBack: () => void;
}) {
  const relativeExpiry = formatRelativeExpiration(meta.expires_at);
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
          burn-after-read paste
        </p>
        <p className="text-xs text-muted-foreground">
          This paste will be permanently destroyed when revealed.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Reveal this paste?</CardTitle>
          <CardDescription>
            Revealing will permanently destroy this paste and any attachments.
            This cannot be undone.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap gap-2">
            <Badge variant="outline" className="text-muted-foreground">
              expires {relativeExpiry}
            </Badge>
            {meta.attachment_count > 0 && (
              <Badge variant="outline" className="text-muted-foreground">
                {meta.attachment_count} attachment
                {meta.attachment_count !== 1 ? "s" : ""}
              </Badge>
            )}
          </div>
          {revealError && (
            <p className="text-sm text-destructive">{revealError}</p>
          )}
          <div className="flex flex-wrap gap-3">
            <Button
              variant="destructive"
              onClick={onReveal}
              disabled={isRevealing}
            >
              {isRevealing ? "Revealing..." : "Reveal"}
            </Button>
            <Button variant="outline" onClick={onBack} disabled={isRevealing}>
              Back
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
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
  const [meta, setMeta] = useState<GetPasteMetaResponse | null>(null);
  const [paste, setPaste] = useState<GetPasteResponse | null>(null);
  const [decrypted, setDecrypted] = useState<{
    title: string;
    body: string;
  } | null>(null);
  const [hashKey, setHashKey] = useState<string | null>(() => getKeyFromHash());
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [isRevealing, setIsRevealing] = useState(false);
  const [revealError, setRevealError] = useState<string | null>(null);

  useEffect(() => {
    const update = () => setHashKey(getKeyFromHash());
    window.addEventListener("hashchange", update);
    return () => window.removeEventListener("hashchange", update);
  }, []);

  useEffect(() => {
    setStatus("loading");
    setMeta(null);
    setPaste(null);
    setDecrypted(null);
    setRevealError(null);

    api
      .getPasteMeta(id)
      .then((data) => {
        setMeta(data);
        if (data.burn_after_read) {
          setStatus("confirm_required");
        } else {
          return api.getPaste(id).then((full) => {
            setPaste(full);
          });
        }
      })
      .catch((err) => {
        const apiErr = getApiError(err);
        if (apiErr?.code === "NOT_FOUND") {
          setStatus("not_found");
        } else if (apiErr?.code === "RATE_LIMITED") {
          setStatus("rate_limited");
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

  if (status === "confirm_required" && meta) {
    return (
      <BurnConfirmInterstitial
        meta={meta}
        isLoggedIn={isLoggedIn}
        isRevealing={isRevealing}
        revealError={revealError}
        onReveal={async () => {
          setRevealError(null);
          setIsRevealing(true);
          try {
            const data = await api.consumePaste(id);
            setPaste(data);
          } catch (err) {
            const apiErr = getApiError(err);
            if (apiErr?.code === "NOT_FOUND") {
              setStatus("not_found");
            } else if (apiErr?.code === "RATE_LIMITED") {
              setStatus("rate_limited");
            } else {
              setRevealError("Failed to reveal paste. It may have expired.");
            }
          } finally {
            setIsRevealing(false);
          }
        }}
        onBack={() => {
          if (isLoggedIn) {
            navigate({ to: "/dashboard", search: { tab: "pastes" } });
          } else {
            navigate({ to: "/" });
          }
        }}
      />
    );
  }

  if (status !== "ready") {
    return (
      <ErrorState
        status={
          status as Exclude<
            Status,
            "loading" | "ready" | "confirm_required"
          >
        }
      />
    );
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

        <CardContent className="pt-4 space-y-4">
          <PasteBody
            body={decrypted!.body}
            language={paste!.language_id ?? "plaintext"}
          />
          {(paste!.attachments?.length ?? 0) > 0 && (
            <AttachmentList
              attachments={paste!.attachments ?? []}
              pasteKey={
                hashKey
                  ? getPasteKeyFromHash(hashKey)
                  : vaultKey
                    ? getPasteKeyFromVault(
                        vaultKey,
                        paste!.encrypted_paste_key_ciphertext,
                        paste!.encrypted_paste_key_nonce
                      )
                    : new Uint8Array()
              }
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
