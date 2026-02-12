import { createFileRoute, Navigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import { z } from "zod";
import { useAuth } from "@/lib/auth";
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
    return (
      <div className="mx-auto max-w-2xl px-4 py-10 space-y-6">
        <div className="h-4 w-32 bg-muted/50 animate-pulse" />
        <div className="h-80 w-full border border-border bg-muted/30 animate-pulse" />
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" />;
  }

  return <NewPasteForm />;
}

function NewPasteForm() {
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
      // TODO: encrypt + upload paste
      console.log("new paste", value);
    },
  });

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
            {/* ── Title + Expiration row ── */}
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

            {/* ── Burn after reading ── */}
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

            {/* ── Body ── */}
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

            {/* ── E2EE notice ── */}
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
              children={([canSubmit, isSubmitting]) => (
                <Button
                  type="submit"
                  disabled={!canSubmit}
                  className="w-full"
                  size="lg"
                >
                  {isSubmitting ? "Encrypting & uploading..." : "Create paste"}
                </Button>
              )}
            />
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
