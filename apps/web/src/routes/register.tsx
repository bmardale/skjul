import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useForm } from "@tanstack/react-form";
import { useState } from "react";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Field, FieldLabel, FieldError } from "@/components/ui/field";
import { HugeiconsIcon } from "@hugeicons/react";
import { Alert02Icon } from "@hugeicons/core-free-icons";
import { generateRegistrationData, deriveLoginKeys, decryptVaultKey } from "@/lib/crypto";
import { api, getApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

export const Route = createFileRoute("/register")({
  component: Register,
});

const registerSchema = z.object({
  username: z
    .string()
    .min(3, "Username must be at least 3 characters")
    .max(32, "Username must be at most 32 characters")
    .regex(/^[a-zA-Z0-9_-]+$/, "Only letters, numbers, _ and -"),
  password: z
    .string()
    .min(8, "Password must be at least 8 characters")
    .max(128, "Password must be at most 128 characters"),
});

function Register() {
  const { setVaultKey, refetchUser } = useAuth();
  const navigate = useNavigate();
  const [formError, setFormError] = useState<string | null>(null);

  const form = useForm({
    defaultValues: {
      username: "",
      password: "",
    },
    validators: {
      onSubmit: registerSchema,
    },
    onSubmit: async ({ value }) => {
      setFormError(null);

      try {
        const registrationData = await generateRegistrationData(value.password);
        await api.register({
          username: value.username,
          auth_key: registrationData.authKey,
          salt: registrationData.salt,
          encrypted_vault_key: registrationData.encryptedVaultKey,
          vault_key_nonce: registrationData.vaultKeyNonce,
        });

        const { masterKey } = await deriveLoginKeys(
          value.password,
          registrationData.salt,
        );
        const vaultKey = await decryptVaultKey(
          masterKey,
          registrationData.encryptedVaultKey,
          registrationData.vaultKeyNonce,
        );

        setVaultKey(vaultKey);
        await refetchUser();
        navigate({ to: "/" });
      } catch (err) {
        const apiErr = getApiError(err);
        if (apiErr?.code === "USERNAME_TAKEN") {
          setFormError("Username is already taken.");
        } else {
          setFormError("Something went wrong. Please try again.");
        }
      }
    },
  });

  return (
    <div className="mx-auto max-w-sm px-4 py-16">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Create an account</CardTitle>
          <CardDescription>
            Register to manage pastes, set expirations, and keep things
            organized.
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
            <form.Field
              name="username"
              children={(field) => {
                const isInvalid =
                  field.state.meta.isTouched && !field.state.meta.isValid;
                return (
                  <Field data-invalid={isInvalid}>
                    <FieldLabel htmlFor={field.name}>Username</FieldLabel>
                    <Input
                      id={field.name}
                      name={field.name}
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(e) => field.handleChange(e.target.value)}
                      aria-invalid={isInvalid}
                      placeholder="satoshi"
                      autoComplete="username"
                    />
                    {isInvalid && (
                      <FieldError errors={field.state.meta.errors} />
                    )}
                  </Field>
                );
              }}
            />

            <form.Field
              name="password"
              children={(field) => {
                const isInvalid =
                  field.state.meta.isTouched && !field.state.meta.isValid;
                return (
                  <Field data-invalid={isInvalid}>
                    <FieldLabel htmlFor={field.name}>Password</FieldLabel>
                    <Input
                      id={field.name}
                      name={field.name}
                      type="password"
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(e) => field.handleChange(e.target.value)}
                      aria-invalid={isInvalid}
                      placeholder="••••••••"
                      autoComplete="new-password"
                    />
                    {isInvalid && (
                      <FieldError errors={field.state.meta.errors} />
                    )}
                  </Field>
                );
              }}
            />

            {formError && (
              <p role="alert" className="text-sm text-destructive text-center">
                {formError}
              </p>
            )}

            <div className="flex gap-2 border border-destructive/30 bg-destructive/5 p-3 text-xs text-muted-foreground">
              <HugeiconsIcon
                icon={Alert02Icon}
                size={16}
                className="shrink-0 text-destructive mt-0.5"
              />
              <p>
                skjul uses end-to-end encryption. If you lose your password,
                your pastes <span className="text-destructive font-medium">cannot be recovered</span>.
                There is no reset.
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
                  {isSubmitting ? "Creating account..." : "Register"}
                </Button>
              )}
            />

            <p className="text-center text-xs text-muted-foreground">
              Already have an account?{" "}
              <Link
                to="/login"
                className="text-primary underline underline-offset-4 hover:text-primary/80"
              >
                Login
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
