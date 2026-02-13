import { lazy, Suspense } from "react";

const PasteBodyHighlighted = lazy(
  () => import("./paste-body-highlighted"),
);

interface PasteBodyProps {
  body: string;
  language?: string;
}

export function PasteBody({ body, language = "plaintext" }: PasteBodyProps) {
  const lang = language?.trim() || "plaintext";
  const useHighlight = lang !== "plaintext" && lang.length > 0;

  if (!useHighlight) {
    return (
      <pre className="whitespace-pre-wrap break-words font-mono text-sm">
        {body}
      </pre>
    );
  }

  return (
    <Suspense
      fallback={
        <pre className="whitespace-pre-wrap break-words font-mono text-sm">
          {body}
        </pre>
      }
    >
      <PasteBodyHighlighted body={body} language={lang} />
    </Suspense>
  );
}
