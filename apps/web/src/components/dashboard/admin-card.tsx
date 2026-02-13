import { useCallback, useState } from "react";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  Cancel01Icon,
  UserCircleIcon,
  Calendar01Icon,
  Attachment01Icon,
  File02Icon,
} from "@hugeicons/core-free-icons";
import {
  api,
  type AdminUserListItem,
  type AdminUserDetail,
} from "@/lib/api";
import { useAsyncData } from "@/lib/hooks/use-async-data";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { DataCard } from "@/components/dashboard/data-card";

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
}

export function AdminCard({ isActive }: { isActive: boolean }) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<AdminUserDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [quotaInput, setQuotaInput] = useState<string>("");
  const [savingQuota, setSavingQuota] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);

  const fetchUsers = useCallback(() => api.adminListUsers(), []);

  const {
    data: users,
    loading,
    error,
    clearError,
    refresh,
  } = useAsyncData<AdminUserListItem[]>(fetchUsers, {
    enabled: isActive,
    initialData: [],
    errorMessage: "Failed to load users.",
  });

  const loadDetail = useCallback(async (id: string) => {
    setDetailLoading(true);
    setLocalError(null);
    try {
      const d = await api.adminGetUser(id);
      setDetail(d);
      setQuotaInput(String(d.invite_quota));
    } catch {
      setLocalError("Failed to load user details.");
      setDetail(null);
    } finally {
      setDetailLoading(false);
    }
  }, []);

  const handleSelectUser = (id: string) => {
    if (selectedId === id) {
      setSelectedId(null);
      setDetail(null);
      return;
    }
    setSelectedId(id);
    void loadDetail(id);
  };

  const handleSaveQuota = async () => {
    if (!selectedId || !detail) return;
    const quota = parseInt(quotaInput, 10);
    if (Number.isNaN(quota) || quota < 0) {
      setLocalError("Invite quota must be a non-negative number.");
      return;
    }
    setSavingQuota(true);
    setLocalError(null);
    try {
      await api.adminUpdateInviteQuota(selectedId, quota);
      setDetail((prev) => (prev ? { ...prev, invite_quota: quota } : null));
      await refresh();
    } catch {
      setLocalError("Failed to update invite quota.");
    } finally {
      setSavingQuota(false);
    }
  };

  const handleRefresh = async () => {
    setLocalError(null);
    clearError();
    await refresh();
    if (selectedId) {
      void loadDetail(selectedId);
    }
  };

  const displayError = localError || error;

  return (
    <DataCard
      title="Admin"
      description="Manage users, view stats, and adjust invite quotas."
      loading={loading}
      error={displayError}
      empty={users.length === 0}
      emptyMessage="No users yet."
      onRefresh={() => void handleRefresh()}
      refreshLabel="Refresh users"
    >
      <div className="space-y-0">
        {users.map((user) => (
          <div key={user.id}>
            <button
              type="button"
              onClick={() => handleSelectUser(user.id)}
              className="w-full rounded-md px-2 py-2 hover:bg-muted/30 transition-colors text-left"
            >
              <div className="flex items-center justify-between gap-4">
                <div className="min-w-0 space-y-0.5">
                  <p className="text-sm font-medium truncate">{user.username}</p>
                  <p className="text-xs text-muted-foreground">
                    {formatDate(user.created_at)} · {user.invite_quota} invite
                    {user.invite_quota !== 1 ? "s" : ""}
                  </p>
                </div>
                <span
                  className={`text-xs transition-transform ${
                    selectedId === user.id ? "rotate-180" : ""
                  }`}
                >
                  ▼
                </span>
              </div>
            </button>

            {selectedId === user.id && (
              <div className="mt-2 rounded-md border border-border bg-muted/20 px-3 py-3 space-y-3">
                {detailLoading ? (
                  <div className="h-24 animate-pulse bg-muted/30 rounded" />
                ) : detail ? (
                  <>
                    <div className="flex items-center gap-3">
                      <HugeiconsIcon
                        icon={UserCircleIcon}
                        size={16}
                        className="text-muted-foreground shrink-0"
                      />
                      <div className="min-w-0 space-y-0.5">
                        <p className="text-xs text-muted-foreground">
                          Username
                        </p>
                        <p className="text-sm font-medium">{detail.username}</p>
                        <p className="text-xs font-mono text-muted-foreground break-all">
                          {detail.id}
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-3">
                      <HugeiconsIcon
                        icon={Calendar01Icon}
                        size={16}
                        className="text-muted-foreground shrink-0"
                      />
                      <div className="space-y-0.5">
                        <p className="text-xs text-muted-foreground">
                          Created
                        </p>
                        <p className="text-sm font-medium">
                          {formatDate(detail.created_at)}
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-3">
                      <HugeiconsIcon
                        icon={File02Icon}
                        size={16}
                        className="text-muted-foreground shrink-0"
                      />
                      <div className="space-y-0.5">
                        <p className="text-xs text-muted-foreground">
                          Pastes
                        </p>
                        <p className="text-sm font-medium">
                          {detail.paste_count}
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-3">
                      <HugeiconsIcon
                        icon={Attachment01Icon}
                        size={16}
                        className="text-muted-foreground shrink-0"
                      />
                      <div className="space-y-0.5">
                        <p className="text-xs text-muted-foreground">
                          Total attachment size
                        </p>
                        <p className="text-sm font-medium">
                          {formatBytes(detail.total_attachment_size)}
                        </p>
                      </div>
                    </div>

                    <Separator />

                    <div className="flex items-center gap-2">
                      <label
                        htmlFor="invite-quota"
                        className="text-xs text-muted-foreground whitespace-nowrap"
                      >
                        Invite quota
                      </label>
                      <Input
                        id="invite-quota"
                        type="number"
                        min={0}
                        value={quotaInput}
                        onChange={(e) => setQuotaInput(e.target.value)}
                        className="w-20 h-8 text-sm"
                      />
                      <Button
                        variant="outline"
                        size="xs"
                        disabled={savingQuota}
                        onClick={() => void handleSaveQuota()}
                      >
                        {savingQuota ? "Saving..." : "Save"}
                      </Button>
                    </div>

                    <div className="pt-2">
                      <DeleteUserButton
                        userId={user.id}
                        username={user.username}
                        onDeleted={() => {
                          setSelectedId(null);
                          setDetail(null);
                          void refresh();
                        }}
                      />
                    </div>
                  </>
                ) : null}
              </div>
            )}
          </div>
        ))}
      </div>
    </DataCard>
  );
}

function DeleteUserButton({
  userId,
  username,
  onDeleted,
}: {
  userId: string;
  username: string;
  onDeleted: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleDelete = async () => {
    setDeleting(true);
    setError(null);
    try {
      await api.adminDeleteUser(userId);
      setOpen(false);
      onDeleted();
    } catch {
      setError("Failed to delete user.");
      setDeleting(false);
    }
  };

  return (
    <AlertDialog
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen);
        if (!nextOpen) setError(null);
      }}
    >
      <AlertDialogTrigger
        render={
          <Button variant="destructive" size="xs">
            <HugeiconsIcon icon={Cancel01Icon} size={12} />
            Delete user
          </Button>
        }
      />
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete user &quot;{username}&quot;?</AlertDialogTitle>
          <AlertDialogDescription>
            This will permanently delete the account and all associated data
            (pastes, attachments, sessions). This action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter className="flex-col items-stretch gap-3">
          {error && (
            <p className="text-xs text-destructive order-last w-full">
              {error}
            </p>
          )}
          <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={deleting}
          >
            {deleting ? "Deleting..." : "Delete user"}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
