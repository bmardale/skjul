const DATE_FORMAT: Intl.DateTimeFormatOptions = {
  year: "numeric",
  month: "short",
  day: "numeric",
};

const NEVER_EXPIRES_YEAR = 2100;

export function formatDate(dateString: string): string {
  const date = new Date(dateString);
  if (date.getFullYear() >= NEVER_EXPIRES_YEAR) {
    return "never";
  }
  return date.toLocaleDateString(undefined, DATE_FORMAT);
}

export function DateRange({
  created,
  expires,
}: {
  created: string;
  expires: string;
}) {
  return (
    <div className="flex gap-3 text-xs text-muted-foreground">
      <span>Created {formatDate(created)}</span>
      <span>Expires {formatDate(expires)}</span>
    </div>
  );
}
