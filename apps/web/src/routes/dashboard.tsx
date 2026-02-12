import { useCallback, useEffect, useState } from "react";
import { createFileRoute, Navigate, useNavigate } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth";
import { api, type SessionResponse } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  Cancel01Icon,
  Refresh01Icon,
  UserCircleIcon,
  Calendar01Icon,
} from "@hugeicons/core-free-icons";
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogAction,
  AlertDialogCancel,
} from "@/components/ui/alert-dialog";

export const Route = createFileRoute("/dashboard")({
  component: Dashboard,
});

function Dashboard() {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-10 space-y-6">
        <div className="h-4 w-32 rounded bg-muted/50 animate-pulse" />
        <div className="h-40 w-full rounded border border-border bg-muted/30 animate-pulse" />
        <div className="h-56 w-full rounded border border-border bg-muted/30 animate-pulse" />
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" />;
  }

  return <DashboardContent />;
}

function DashboardContent() {
  const { user } = useAuth();

  return (
    <div className="mx-auto max-w-2xl px-4 py-10 space-y-6">
      <div className="space-y-1">
        <p className="text-sm font-medium font-mono tracking-tight">
          <span className="text-muted-foreground">$ </span>
          <span className="text-foreground">dashboard</span>
        </p>
        <p className="text-xs text-muted-foreground">
          Manage your account and sessions.
        </p>
      </div>

      {/* ── Account ── */}
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
              <p className="text-sm font-medium">{user!.username}</p>
              <p className="text-xs font-mono text-muted-foreground break-all">
                {user!.user_id}
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
                {new Date(user!.created_at).toLocaleDateString(undefined, {
                  year: "numeric",
                  month: "long",
                  day: "numeric",
                })}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* ── Sessions ── */}
      <SessionsCard />

      {/* ── Danger zone ── */}
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

function SessionsCard() {
  const [sessions, setSessions] = useState<SessionResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [revokingIds, setRevokingIds] = useState<Set<string>>(new Set());

  const loadSessions = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.listSessions();
      setSessions(data);
    } catch {
      setError("Failed to load sessions.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadSessions();
  }, [loadSessions]);

  const handleRevoke = async (id: string) => {
    setRevokingIds((prev) => new Set(prev).add(id));
    try {
      await api.revokeSession(id);
      setSessions((prev) => prev.filter((s) => s.id !== id));
    } catch {
      setError("Failed to revoke session.");
    } finally {
      setRevokingIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          Sessions
          <Button
            variant="ghost"
            size="icon-xs"
            onClick={loadSessions}
            disabled={loading}
            aria-label="Refresh sessions"
            title="Refresh sessions"
          >
            <HugeiconsIcon icon={Refresh01Icon} size={14} />
          </Button>
        </CardTitle>
        <CardDescription>
          Active sessions on this account. Revoke any you don't recognize.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="space-y-3">
            {[1, 2].map((i) => (
              <div
                key={i}
                className="h-12 w-full bg-muted/30 animate-pulse"
              />
            ))}
          </div>
        ) : error ? (
          <p className="text-xs text-destructive">{error}</p>
        ) : sessions.length === 0 ? (
          <p className="text-xs text-muted-foreground">No active sessions.</p>
        ) : (
          <div className="space-y-0">
            {sessions.map((session, i) => (
              <div key={session.id}>
                {i > 0 && <Separator className="my-3" />}
                <div className="rounded-md px-2 py-2 hover:bg-muted/30 transition-colors">
                <div className="flex items-center justify-between gap-4">
                  <div className="min-w-0 space-y-1">
                    <div className="flex items-center gap-2">
                      <p className="text-xs font-medium font-mono truncate">
                        {session.id}
                      </p>
                      {session.current && (
                        <Badge variant="outline" className="text-primary">
                          current
                        </Badge>
                      )}
                    </div>
                    <div className="flex gap-3 text-xs text-muted-foreground">
                      <span>
                        Created{" "}
                        {new Date(session.created_at).toLocaleDateString()}
                      </span>
                      <span>
                        Expires{" "}
                        {new Date(session.expires_at).toLocaleDateString()}
                      </span>
                    </div>
                  </div>
                  {!session.current && (
                    <Button
                      variant="destructive"
                      size="xs"
                      disabled={revokingIds.has(session.id)}
                      onClick={() => handleRevoke(session.id)}
                    >
                      <HugeiconsIcon icon={Cancel01Icon} size={12} />
                      {revokingIds.has(session.id) ? "Revoking…" : "Revoke"}
                    </Button>
                  )}
                </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
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
        <AlertDialog open={open} onOpenChange={setOpen}>
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
            {error && (
              <p className="text-xs text-destructive">{error}</p>
            )}
            <AlertDialogFooter>
              <AlertDialogCancel disabled={deleting}>
                Cancel
              </AlertDialogCancel>
              <AlertDialogAction
                variant="destructive"
                onClick={handleDelete}
                disabled={deleting}
              >
                {deleting ? "Deleting…" : "Delete account"}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </div>
  );
}
