import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  Calendar01Icon,
  Cancel01Icon,
  UserCircleIcon,
} from "@hugeicons/core-free-icons";
import { api, type MeResponse } from "@/lib/api";
import { useAuth } from "@/lib/auth";
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
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export function AccountCard({ user }: { user: MeResponse }) {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Account</CardTitle>
          <CardDescription>Your identity on this instance.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center gap-3">
            <HugeiconsIcon
              icon={UserCircleIcon}
              size={16}
              className="text-muted-foreground shrink-0"
            />
            <div className="space-y-0.5 min-w-0">
              <p className="text-xs text-muted-foreground">Username</p>
              <p className="text-sm font-medium">{user.username}</p>
              <p className="text-xs font-mono text-muted-foreground break-all">
                {user.user_id}
              </p>
            </div>
          </div>
          <Separator />
          <div className="flex items-center gap-3">
            <HugeiconsIcon
              icon={Calendar01Icon}
              size={16}
              className="text-muted-foreground shrink-0"
            />
            <div className="space-y-0.5">
              <p className="text-xs text-muted-foreground">Member since</p>
              <p className="text-sm font-medium">
                {new Date(user.created_at).toLocaleDateString(undefined, {
                  year: "numeric",
                  month: "long",
                  day: "numeric",
                })}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="border-destructive/30">
        <CardHeader>
          <CardTitle className="text-destructive">Danger zone</CardTitle>
          <CardDescription>
            Irreversible actions on your account.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <DeleteAccountSection />
        </CardContent>
      </Card>
    </div>
  );
}

function DeleteAccountSection() {
  const { logout } = useAuth();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleDelete = async () => {
    setDeleting(true);
    setError(null);
    try {
      await api.deleteAccount();
      await logout();
      navigate({ to: "/login" });
    } catch {
      setError("Failed to delete account.");
      setDeleting(false);
    }
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-4">
        <div className="space-y-0.5">
          <p className="text-sm font-medium">Delete account</p>
          <p className="text-xs text-muted-foreground">
            Permanently delete your account and all associated data.
          </p>
        </div>
        <AlertDialog
          open={open}
          onOpenChange={(nextOpen) => {
            setOpen(nextOpen);
            if (!nextOpen) setError(null);
          }}
        >
          <AlertDialogTrigger
            render={<Button variant="destructive" size="xs" />}
          >
            <HugeiconsIcon icon={Cancel01Icon} size={12} />
            Delete
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete account?</AlertDialogTitle>
              <AlertDialogDescription>
                This will permanently delete your account, all your secrets, and
                revoke all sessions. This action cannot be undone.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter className="flex-col items-stretch gap-3">
              {error && (
                <p className="text-xs text-destructive order-last w-full">
                  {error}
                </p>
              )}
              <AlertDialogCancel disabled={deleting}>
                Cancel
              </AlertDialogCancel>
              <AlertDialogAction
                variant="destructive"
                onClick={handleDelete}
                disabled={deleting}
              >
                {deleting ? "Deleting..." : "Delete account"}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </div>
  );
}
