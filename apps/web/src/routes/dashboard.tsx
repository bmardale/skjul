import { createFileRoute, Navigate, useNavigate } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth";
import { AccountCard } from "@/components/dashboard/account-card";
import { SessionsCard } from "@/components/dashboard/sessions-card";
import { PastesCard } from "@/components/dashboard/pastes-card";
import { PageSkeleton } from "@/components/ui/page-skeleton";

const TAB_OPTIONS = ["account", "sessions", "pastes"] as const;
type Tab = (typeof TAB_OPTIONS)[number];

export const Route = createFileRoute("/dashboard")({
  validateSearch: (search: Record<string, unknown>) => {
    const tab = search.tab as string | undefined;
    if (TAB_OPTIONS.includes(tab as Tab)) {
      return { tab: tab as Tab };
    }
    return { tab: "account" as Tab };
  },
  component: Dashboard,
});

function Dashboard() {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <PageSkeleton
        blocks={[
          "h-40 w-full rounded border border-border bg-muted/30 animate-pulse",
          "h-56 w-full rounded border border-border bg-muted/30 animate-pulse",
        ]}
      />
    );
  }

  if (!user) {
    return <Navigate to="/login" />;
  }

  return <DashboardContent />;
}

function DashboardContent() {
  const { user } = useAuth();
  const navigate = useNavigate({ from: Route.fullPath });
  const { tab } = Route.useSearch();

  if (!user) return null;

  return (
    <div className="mx-auto max-w-2xl px-4 py-10 space-y-6">
      <div className="space-y-1">
        <p className="text-sm font-medium font-mono tracking-tight">
          <span className="text-muted-foreground">$ </span>
          <span className="text-foreground">dashboard</span>
        </p>
        <p className="text-xs text-muted-foreground">
          Manage your account, sessions, and pastes.
        </p>
      </div>

      <div
        role="tablist"
        aria-label="Dashboard sections"
        className="flex gap-1 border-b border-border"
      >
        {TAB_OPTIONS.map((option) => {
          const selected = tab === option;
          return (
            <button
              key={option}
              type="button"
              role="tab"
              aria-selected={selected}
              tabIndex={selected ? 0 : -1}
              aria-controls={`dashboard-panel-${option}`}
              id={`dashboard-tab-${option}`}
              onClick={() => navigate({ search: { tab: option } })}
              className={`px-3 py-1.5 text-xs font-medium transition-colors ${
                selected
                  ? "text-foreground border-b-2 border-foreground -mb-px"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {option.charAt(0).toUpperCase() + option.slice(1)}
            </button>
          );
        })}
      </div>

      <section
        role="tabpanel"
        id="dashboard-panel-account"
        aria-labelledby="dashboard-tab-account"
        hidden={tab !== "account"}
      >
        <AccountCard user={user} />
      </section>
      <section
        role="tabpanel"
        id="dashboard-panel-sessions"
        aria-labelledby="dashboard-tab-sessions"
        hidden={tab !== "sessions"}
      >
        <SessionsCard isActive={tab === "sessions"} />
      </section>
      <section
        role="tabpanel"
        id="dashboard-panel-pastes"
        aria-labelledby="dashboard-tab-pastes"
        hidden={tab !== "pastes"}
      >
        <PastesCard isActive={tab === "pastes"} />
      </section>
    </div>
  );
}