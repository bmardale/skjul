import { useCallback, useState } from "react";
import { HugeiconsIcon } from "@hugeicons/react";
import { Add01Icon } from "@hugeicons/core-free-icons";
import { api, type ListInvitationsResponse } from "@/lib/api";
import { useAsyncData } from "@/lib/hooks/use-async-data";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
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

export function InvitationsCard({ isActive }: { isActive: boolean }) {
  const [generating, setGenerating] = useState(false);
  const [localError, setLocalError] = useState<string | null>(null);
  const [copiedCode, setCopiedCode] = useState<string | null>(null);

  const fetchInvites = useCallback(() => api.listInvites(), []);

  const {
    data,
    loading,
    error,
    clearError,
    refresh,
  } = useAsyncData<ListInvitationsResponse>(fetchInvites, {
    enabled: isActive,
    initialData: { remaining_quota: 0, invitations: [] },
    errorMessage: "Failed to load invitations.",
  });

  const handleGenerate = async () => {
    setLocalError(null);
    setGenerating(true);
    try {
      const { code } = await api.generateInvite();
      await navigator.clipboard.writeText(code);
      setCopiedCode(code);
      setTimeout(() => setCopiedCode(null), 2000);
      await refresh();
    } catch {
      setLocalError("Failed to generate invite.");
    } finally {
      setGenerating(false);
    }
  };

  const handleRefresh = async () => {
    setLocalError(null);
    clearError();
    await refresh();
  };

  const displayError = localError || error;

  return (
    <DataCard
      title="Invitations"
      description="Generate invite codes for new users. Share a code with someone to let them register."
      loading={loading}
      error={displayError}
      empty={false}
      emptyMessage=""
      onRefresh={() => void handleRefresh()}
      refreshLabel="Refresh invitations"
    >
      <div className="space-y-3">
        <div className="flex items-center justify-between gap-2">
          <p className="text-xs text-muted-foreground">
            {data.remaining_quota} invite{data.remaining_quota !== 1 ? "s" : ""} remaining
          </p>
          <Button
            variant="outline"
            size="xs"
            disabled={generating || data.remaining_quota <= 0}
            onClick={() => void handleGenerate()}
          >
            <HugeiconsIcon icon={Add01Icon} size={12} />
            {generating ? "Generating..." : "Generate invite"}
          </Button>
        </div>

        {data.invitations.length > 0 ? (
          <>
            <Separator />
            <div className="space-y-0">
              {data.invitations.map((inv, i) => (
                <div key={inv.id}>
                  {i > 0 && <Separator className="my-3" />}
                  <div className="rounded-md px-2 py-2 hover:bg-muted/30 transition-colors">
                    <div className="flex items-center justify-between gap-4">
                      <div className="min-w-0 space-y-1">
                        <div className="flex items-center gap-2">
                          <code className="text-xs font-mono truncate">
                            {inv.code}
                          </code>
                          {inv.used ? (
                            <Badge variant="outline" className="text-muted-foreground">
                              used
                            </Badge>
                          ) : (
                            <Badge variant="outline" className="text-primary">
                              unused
                            </Badge>
                          )}
                        </div>
                        <p className="text-xs text-muted-foreground">
                          Created {formatDate(inv.created_at)}
                          {inv.used_at && ` · Used ${formatDate(inv.used_at)}`}
                        </p>
                      </div>
                      {!inv.used && (
                        <Button
                          variant="ghost"
                          size="xs"
                          onClick={() => {
                            navigator.clipboard.writeText(inv.code);
                            setCopiedCode(inv.code);
                            setTimeout(() => setCopiedCode(null), 2000);
                          }}
                        >
                          {copiedCode === inv.code ? "Copied!" : "Copy"}
                        </Button>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </>
        ) : (
          <p className="text-xs text-muted-foreground">No invitations yet.</p>
        )}
      </div>
    </DataCard>
  );
}
