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
import { api, getApiError } from "@/lib/api";
import { deriveLoginKeys, decryptVaultKey } from "@/lib/crypto";
import { useAuth } from "@/lib/auth";

export const Route = createFileRoute("/login")({
  component: Login,
});

const loginSchema = z.object({
  username: z.string().min(1, "Username is required"),
  password: z.string().min(1, "Password is required"),
});

function Login() {
  const { setVaultKey, refetchUser } = useAuth();
  const navigate = useNavigate();
  const [formError, setFormError] = useState<string | null>(null);

  const form = useForm({
    defaultValues: {
      username: "",
      password: "",
    },
    validators: {
      onSubmit: loginSchema,
    },
    onSubmit: async ({ value }) => {
      setFormError(null);

      try {
        // 1. Get challenge (salt only)
        const challenge = await api.loginChallenge({
          username: value.username,
        });

        // 2. Derive masterKey + authKey from password + salt
        const { masterKey, authKey } = await deriveLoginKeys(
          value.password,
          challenge.salt,
        );

        // 3. Authenticate with derived auth key (sets session cookie)
        await api.login({
          username: value.username,
          auth_key: authKey,
        });

        // 4. Fetch encrypted vault key from /me (now authenticated)
        const me = await api.me();

        // 5. Decrypt vault key with masterKey still in memory
        const vaultKey = await decryptVaultKey(
          masterKey,
          me.encrypted_vault_key,
          me.vault_key_nonce,
        );

        setVaultKey(vaultKey);
        await refetchUser();

        // 6. Redirect to dashboard
        navigate({ to: "/" });
      } catch (err) {
        const apiErr = getApiError(err);
        if (apiErr?.code === "INVALID_CREDENTIALS") {
          setFormError("Invalid username or password.");
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
          <CardTitle className="text-base">Welcome back</CardTitle>
          <CardDescription>
            Sign in to access your encrypted pastes.
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
                      autoComplete="current-password"
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
                  {isSubmitting ? "Signing in..." : "Login"}
                </Button>
              )}
            />

            <p className="text-center text-xs text-muted-foreground">
              Don't have an account?{" "}
              <Link
                to="/register"
                className="text-primary underline underline-offset-4 hover:text-primary/80"
              >
                Register
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
