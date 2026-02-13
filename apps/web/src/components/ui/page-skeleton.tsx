interface PageSkeletonProps {
  blocks?: string[];
}

const DEFAULT_BLOCKS = [
  "h-80 w-full rounded border border-border bg-muted/30 animate-pulse",
];

export function PageSkeleton({ blocks = DEFAULT_BLOCKS }: PageSkeletonProps) {
  return (
    <div className="mx-auto max-w-2xl px-4 py-10 space-y-6">
      <div className="h-4 w-32 rounded bg-muted/50 animate-pulse" />
      {blocks.map((blockClassName, i) => (
        <div key={`${blockClassName}-${i}`} className={blockClassName} />
      ))}
    </div>
  );
}
