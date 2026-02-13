import { type ReactNode } from "react";
import { HugeiconsIcon } from "@hugeicons/react";
import { Refresh01Icon } from "@hugeicons/core-free-icons";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

function LoadCardSkeleton({ rows = 2 }: { rows?: number }) {
  return (
    <div className="space-y-3">
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="h-12 w-full bg-muted/30 animate-pulse" />
      ))}
    </div>
  );
}

interface DataCardProps {
  title: string;
  description: string;
  loading: boolean;
  error: string | null;
  empty: boolean;
  emptyMessage: string;
  onRefresh: () => void;
  refreshLabel?: string;
  children: ReactNode;
}

export function DataCard({
  title,
  description,
  loading,
  error,
  empty,
  emptyMessage,
  onRefresh,
  refreshLabel,
  children,
}: DataCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          {title}
          <Button
            variant="ghost"
            size="icon-xs"
            onClick={onRefresh}
            disabled={loading}
            aria-label={refreshLabel ?? `Refresh ${title.toLowerCase()}`}
            title={refreshLabel ?? `Refresh ${title.toLowerCase()}`}
          >
            <HugeiconsIcon icon={Refresh01Icon} size={14} />
          </Button>
        </CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        {loading ? (
          <LoadCardSkeleton />
        ) : error ? (
          <p className="text-xs text-destructive">{error}</p>
        ) : empty ? (
          <p className="text-xs text-muted-foreground">{emptyMessage}</p>
        ) : (
          children
        )}
      </CardContent>
    </Card>
  );
}
