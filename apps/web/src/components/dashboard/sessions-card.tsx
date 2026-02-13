import { useCallback, useState } from "react";
import { HugeiconsIcon } from "@hugeicons/react";
import { Cancel01Icon } from "@hugeicons/core-free-icons";
import { api, type SessionResponse } from "@/lib/api";
import { useAsyncData } from "@/lib/hooks/use-async-data";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { DataCard } from "@/components/dashboard/data-card";
import { DateRange } from "@/components/dashboard/date-range";

export function SessionsCard({ isActive }: { isActive: boolean }) {
  const [revokingId, setRevokingId] = useState<string | null>(null);
  const [localError, setLocalError] = useState<string | null>(null);
  const fetchSessions = useCallback(() => api.listSessions(), []);

  const { data: sessions, setData: setSessions, loading, error, clearError, refresh } =
    useAsyncData<SessionResponse[]>(fetchSessions, {
      enabled: isActive,
      initialData: [],
      errorMessage: "Failed to load sessions.",
    });

  const handleRevoke = async (id: string) => {
    setLocalError(null);
    setRevokingId(id);
    try {
      await api.revokeSession(id);
      setSessions((prev) => prev.filter((session) => session.id !== id));
    } catch {
      setLocalError("Failed to revoke session.");
    } finally {
      setRevokingId(null);
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
      title="Sessions"
      description="Active sessions on this account. Revoke any you don't recognize."
      loading={loading}
      error={displayError}
      empty={sessions.length === 0}
      emptyMessage="No active sessions."
      onRefresh={() => void handleRefresh()}
      refreshLabel="Refresh sessions"
    >
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
                  <DateRange
                    created={session.created_at}
                    expires={session.expires_at}
                  />
                </div>
                {!session.current && (
                  <Button
                    variant="destructive"
                    size="xs"
                    disabled={revokingId === session.id}
                    onClick={() => void handleRevoke(session.id)}
                  >
                    <HugeiconsIcon icon={Cancel01Icon} size={12} />
                    {revokingId === session.id ? "Revoking..." : "Revoke"}
                  </Button>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </DataCard>
  );
}
