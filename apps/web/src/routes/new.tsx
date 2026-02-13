import { useState, useRef } from "react";
import { createFileRoute, Navigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import { z } from "zod";
import { bytesToHex } from "@noble/hashes/utils.js";
import { useAuth } from "@/lib/auth";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Separator } from "@/components/ui/separator";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import { Field, FieldLabel, FieldDescription, FieldError } from "@/components/ui/field";
import { Checkbox } from "@/components/ui/checkbox";
import { HugeiconsIcon } from "@hugeicons/react";
import { Alert02Icon } from "@hugeicons/core-free-icons";
import {
  createPaste,
  encryptFile,
  encryptFilename,
} from "@/lib/crypto";

const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB
const MAX_ATTACHMENTS = 5;
import { PageSkeleton } from "../components/ui/page-skeleton";

export const Route = createFileRoute("/new")({
  component: NewPaste,
});

const expirationOptions = [
  { value: "30m", label: "30 minutes" },
  { value: "1h", label: "1 hour" },
  { value: "1d", label: "1 day" },
  { value: "7d", label: "7 days" },
  { value: "30d", label: "30 days" },
  { value: "never", label: "Never" },
] as const;

type ExpirationValue = (typeof expirationOptions)[number]["value"];

const newPasteSchema = z.object({
  title: z
    .string()
    .min(1, "Title is required")
    .max(128, "Title must be at most 128 characters"),
  expiration: z.enum(["30m", "1h", "1d", "7d", "30d", "never"]),
  burn_after_reading: z.boolean(),
  body: z
    .string()
    .min(1, "Body cannot be empty")
    .max(100_000, "Body must be at most 100,000 characters"),
});

function NewPaste() {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return <PageSkeleton />;
  }

  if (!user) {
    return <Navigate to="/login" />;
  }

  return <NewPasteForm />;
}

interface CreatedPaste {
  id: string;
  shareUrl: string;
  title: string;
  body: string;
}

interface SelectedFile {
  file: File;
  error?: string;
}

function NewPasteForm() {
  const { vaultKey } = useAuth();
  const [created, setCreated] = useState<CreatedPaste | null>(null);
  const [copied, setCopied] = useState(false);
  const [selectedFiles, setSelectedFiles] = useState<SelectedFile[]>([]);
  const [uploadProgress, setUploadProgress] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const form = useForm({
    defaultValues: {
      title: "",
      expiration: "1d" as ExpirationValue,
      burn_after_reading: false,
      body: "",
    },
    validators: {
      onSubmit: newPasteSchema,
    },
    onSubmit: async ({ value }) => {
      if (!vaultKey) {
        throw new Error("Vault key not found");
      }

      const paste = await createPaste(value.title, value.body, vaultKey);
      const res = await api.createPaste({
        encrypted_title_ciphertext: bytesToHex(paste.encryptedTitleCiphertext),
        encrypted_title_nonce: bytesToHex(paste.encryptedTitleNonce),
        encrypted_body_ciphertext: bytesToHex(paste.encryptedBodyCiphertext),
        encrypted_body_nonce: bytesToHex(paste.encryptedBodyNonce),
        encrypted_paste_key_ciphertext: bytesToHex(paste.encryptedPasteKeyCiphertext),
        encrypted_paste_key_nonce: bytesToHex(paste.encryptedPasteKeyNonce),
        expiration: value.expiration,
        burn_after_reading: value.burn_after_reading,
      });

      const validFiles = selectedFiles.filter((f) => !f.error);
      for (let i = 0; i < validFiles.length; i++) {
        setUploadProgress(`Uploading attachment ${i + 1} of ${validFiles.length}...`);
        const { file } = validFiles[i];
        const bytes = new Uint8Array(await file.arrayBuffer());
        const { ciphertext, nonce } = encryptFile(bytes, paste.pasteKey);
        const { ciphertext: filenameCiphertext, nonce: filenameNonce } = encryptFilename(
          file.name,
          paste.pasteKey
        );
        const mimeType = file.type || "application/octet-stream";
        const { ciphertext: mimeCiphertext, nonce: mimeNonce } = encryptFilename(
          mimeType,
          paste.pasteKey
        );

        const attachmentRes = await api.createAttachment(res.id, {
          encrypted_size: ciphertext.length,
          filename_ciphertext: bytesToHex(filenameCiphertext),
          filename_nonce: bytesToHex(filenameNonce),
          content_nonce: bytesToHex(nonce),
          mime_ciphertext: bytesToHex(mimeCiphertext),
          mime_nonce: bytesToHex(mimeNonce),
        });

        await api.uploadToPresignedUrl(attachmentRes.upload_url, ciphertext);
      }
      setUploadProgress(null);

      const shareUrl = `${window.location.origin}/pastes/${res.id}#key=${paste.pasteKeyBase64Url}`;
      setCreated({
        id: res.id,
        shareUrl,
        title: value.title,
        body: value.body,
      });
    },
  });

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? []);
    e.target.value = "";
    if (files.length === 0) return;

    setSelectedFiles((prev) => {
      const next = [...prev];
      for (const file of files) {
        if (next.length >= MAX_ATTACHMENTS) break;
        const error =
          file.size > MAX_FILE_SIZE
            ? `File exceeds 10MB limit`
            : next.some((f) => f.file.name === file.name)
              ? "Duplicate filename"
              : undefined;
        next.push({ file, error });
      }
      return next;
    });
  };

  const removeFile = (index: number) => {
    setSelectedFiles((prev) => prev.filter((_, i) => i !== index));
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    if (fileInputRef.current && e.dataTransfer.files.length) {
      const dt = new DataTransfer();
      Array.from(e.dataTransfer.files).forEach((f) => dt.items.add(f));
      fileInputRef.current.files = dt.files;
      fileInputRef.current.dispatchEvent(new Event("change", { bubbles: true }));
    }
  };

  const handleDragOver = (e: React.DragEvent) => e.preventDefault();

  if (created) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-10 space-y-6">
        <div className="space-y-1">
          <p className="text-sm font-medium">
            <span className="text-muted-foreground">$ </span>
            paste created
          </p>
          <p className="text-xs text-muted-foreground">
            Share the link below. The decryption key is embedded in the URL
            fragment.
          </p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Share link</CardTitle>
            <CardDescription>
              Anyone with this link can decrypt and read the paste.
            </CardDescription>
          </CardHeader>

          <CardContent className="space-y-4">
            <div className="flex gap-2">
              <Input readOnly value={created.shareUrl} className="font-mono text-xs" />
              <Button
                type="button"
                variant="outline"
                className="shrink-0"
                onClick={() => {
                  navigator.clipboard.writeText(created.shareUrl);
                  setCopied(true);
                  setTimeout(() => setCopied(false), 2000);
                }}
              >
                {copied ? "Copied!" : "Copy"}
              </Button>
            </div>

            <Separator />

            <div className="space-y-2">
              <p className="text-sm font-medium">{created.title}</p>
              <pre className="whitespace-pre-wrap break-words font-mono text-sm text-muted-foreground">
                {created.body}
              </pre>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-10 space-y-6">
      <div className="space-y-1">
        <p className="text-sm font-medium">
          <span className="text-muted-foreground">$ </span>
          new paste
        </p>
        <p className="text-xs text-muted-foreground">
          Write, encrypt, and share.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Create paste</CardTitle>
          <CardDescription>
            Content is encrypted before it leaves your browser.
          </CardDescription>
        </CardHeader>

        <CardContent>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              form.handleSubmit();
            }}
            className="space-y-4"
          >
            <div className="flex flex-col gap-4 sm:flex-row">
              <form.Field
                name="title"
                children={(field) => {
                  const isInvalid =
                    field.state.meta.isTouched && !field.state.meta.isValid;
                  return (
                    <Field data-invalid={isInvalid} className="flex-1">
                      <FieldLabel htmlFor={field.name}>Title</FieldLabel>
                      <Input
                        id={field.name}
                        name={field.name}
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(e) => field.handleChange(e.target.value)}
                        aria-invalid={isInvalid}
                        placeholder="my-secret-config"
                        autoComplete="off"
                      />
                      {isInvalid && (
                        <FieldError errors={field.state.meta.errors} />
                      )}
                    </Field>
                  );
                }}
              />

              <form.Field
                name="expiration"
                children={(field) => (
                  <Field className="sm:w-40">
                    <FieldLabel htmlFor={field.name}>Expires</FieldLabel>
                    <Select
                      value={field.state.value}
                      onValueChange={(v) => field.handleChange(v as ExpirationValue)}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {expirationOptions.map((opt) => (
                          <SelectItem key={opt.value} value={opt.value}>
                            {opt.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </Field>
                )}
              />
            </div>

            <form.Field
              name="burn_after_reading"
              children={(field) => (
                <Field orientation="horizontal">
                  <Checkbox
                    id={field.name}
                    checked={field.state.value}
                    onCheckedChange={(checked) =>
                      field.handleChange(checked as boolean)
                    }
                  />
                  <label htmlFor={field.name} className="space-y-0.5 cursor-pointer">
                    <p className="text-xs font-medium">Burn after reading</p>
                    <FieldDescription>
                      Destroy this paste after it has been viewed once.
                    </FieldDescription>
                  </label>
                </Field>
              )}
            />

            <form.Field
              name="body"
              children={(field) => {
                const isInvalid =
                  field.state.meta.isTouched && !field.state.meta.isValid;
                return (
                  <Field data-invalid={isInvalid}>
                    <FieldLabel htmlFor={field.name}>Body</FieldLabel>
                    <Textarea
                      id={field.name}
                      name={field.name}
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(e) => field.handleChange(e.target.value)}
                      aria-invalid={isInvalid}
                      placeholder="Paste your content here..."
                      className="min-h-48 font-mono"
                    />
                    {isInvalid && (
                      <FieldError errors={field.state.meta.errors} />
                    )}
                  </Field>
                );
              }}
            />

            <div>
              <FieldLabel>Attachments (max 5, 10MB each)</FieldLabel>
              <input
                ref={fileInputRef}
                type="file"
                multiple
                className="hidden"
                onChange={handleFileSelect}
              />
              <div
                onDrop={handleDrop}
                onDragOver={handleDragOver}
                onClick={() => fileInputRef.current?.click()}
                className="mt-2 flex min-h-20 cursor-pointer flex-col items-center justify-center rounded-md border border-dashed border-muted-foreground/30 bg-muted/30 px-4 py-6 text-sm text-muted-foreground transition-colors hover:bg-muted/50"
              >
                Drop files here or click to select
              </div>
              {selectedFiles.length > 0 && (
                <ul className="mt-2 space-y-2">
                  {selectedFiles.map(({ file, error }, i) => (
                    <li
                      key={i}
                      className="flex items-center justify-between gap-2 rounded-md border bg-muted/30 px-3 py-2 text-sm"
                    >
                      <span className="truncate" title={file.name}>
                        {file.name}
                      </span>
                      <span className="shrink-0 text-muted-foreground">
                        {(file.size / 1024).toFixed(1)} KB
                      </span>
                      {error && (
                        <span className="shrink-0 text-destructive text-xs">
                          {error}
                        </span>
                      )}
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 shrink-0 p-0"
                        onClick={() => removeFile(i)}
                      >
                        ×
                      </Button>
                    </li>
                  ))}
                </ul>
              )}
            </div>

            <div className="flex gap-2 border border-primary/20 bg-primary/5 p-3 text-xs text-muted-foreground">
              <HugeiconsIcon
                icon={Alert02Icon}
                size={16}
                className="shrink-0 text-primary mt-0.5"
              />
              <p>
                This paste will be encrypted client-side. The decryption key is
                embedded in the share link — only people with the link can read
                it.
              </p>
            </div>

            <Separator />

            <form.Subscribe
              selector={(state) => [state.canSubmit, state.isSubmitting]}
              children={([canSubmit, isSubmitting]) => {
                const hasFileErrors = selectedFiles.some((f) => f.error);
                return (
                  <Button
                    type="submit"
                    disabled={!canSubmit || hasFileErrors || uploadProgress !== null}
                    className="w-full"
                    size="lg"
                  >
                    {uploadProgress ??
                      (isSubmitting ? "Encrypting & uploading..." : "Create paste")}
                  </Button>
                );
              }}
            />
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
