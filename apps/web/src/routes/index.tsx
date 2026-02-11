import { createFileRoute, Link } from "@tanstack/react-router";
import { ModeToggle } from "@/components/mode-toggle";
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
  LockIcon,
  ServerStack01Icon,
  UserCircleIcon,
  Share01Icon,
  OpenSourceIcon,
  Github01Icon,
  ArrowRight01Icon,
  CommandLineIcon,
} from "@hugeicons/core-free-icons";

export const Route = createFileRoute("/")({
  component: Index,
});

const features = [
  {
    icon: LockIcon,
    title: "E2E by default",
    description: "Encrypt in the browser. The server stores ciphertext only.",
  },
  {
    icon: ServerStack01Icon,
    title: "Self-hostable",
    description: "Run it with Docker. Keep data on your own infrastructure.",
  },
  {
    icon: UserCircleIcon,
    title: "Accounts",
    description:
      "Sign in to manage pastes, set expirations, and stay organized.",
  },
  {
    icon: Share01Icon,
    title: "Paste sharing",
    description: "Send a link. Recipients decrypt locally.",
  },
  {
    icon: OpenSourceIcon,
    title: "Open source",
    description: "Audit the crypto. Fork it. Ship it.",
  },
];

const steps = [
  "write a paste",
  "encrypt locally (key never leaves your device)",
  "upload ciphertext → share link",
];

function Index() {
  return (
    <div className="min-h-svh bg-background text-foreground">
      <div className="mx-auto max-w-2xl px-4 py-8">
        {/* ── Header ── */}
        <header className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground">$</span>
            <span className="text-sm font-medium tracking-tight">skjul</span>
            <Badge variant="outline">E2E</Badge>
            <Badge variant="outline">self-host</Badge>
          </div>
          <div className="flex items-center gap-1">
            <Button
              variant="outline"
              size="sm"
              render={
                <a
                  href="https://github.com/skjul/skjul"
                  target="_blank"
                  rel="noreferrer"
                />
              }
            >
              <HugeiconsIcon icon={Github01Icon} size={14} />
              GitHub
            </Button>
            <ModeToggle />
          </div>
        </header>

        <Separator className="my-6" />

        {/* ── Hero ── */}
        <section className="space-y-4">
          <div className="flex flex-wrap gap-1.5">
            {["zero-knowledge", "accounts", "open source", "paste sharing"].map(
              (tag) => (
                <Badge key={tag} variant="secondary">
                  {tag}
                </Badge>
              ),
            )}
          </div>

          <h1 className="text-2xl font-medium tracking-tight">
            End-to-end encrypted pastes.
          </h1>
          <p className="text-sm text-muted-foreground">
            Self-hostable pastebin with accounts — the server never sees
            plaintext.
          </p>

          <div className="flex flex-col gap-2 sm:flex-row">
            <Button size="lg" render={<Link to="/" />}>
              <HugeiconsIcon icon={CommandLineIcon} size={16} />
              Create a paste
            </Button>
            <Button variant="outline" size="lg" render={<a href="#quickstart" />}>
              <HugeiconsIcon icon={ServerStack01Icon} size={16} />
              Self-host
            </Button>
          </div>

          <p className="text-xs text-muted-foreground">
            skjul:~$ encrypt → upload → share
          </p>
        </section>

        <Separator className="my-10" />

        {/* ── Features ── */}
        <section className="space-y-4">
          <p className="text-xs text-muted-foreground uppercase tracking-widest">
            Features
          </p>
          <div className="grid gap-3 sm:grid-cols-2">
            {features.map((feature) => (
              <Card key={feature.title} size="sm" className="bg-card/50">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <HugeiconsIcon
                      icon={feature.icon}
                      size={16}
                      className="text-primary"
                    />
                    {feature.title}
                  </CardTitle>
                  <CardDescription>{feature.description}</CardDescription>
                </CardHeader>
              </Card>
            ))}
          </div>
        </section>

        <Separator className="my-10" />

        {/* ── How it works ── */}
        <section className="space-y-4">
          <p className="text-xs text-muted-foreground uppercase tracking-widest">
            How it works
          </p>
          <Card size="sm" className="bg-muted/30">
            <CardContent className="space-y-1.5">
              {steps.map((step, i) => (
                <div
                  key={i}
                  className="flex gap-3 text-sm text-muted-foreground"
                >
                  <span className="text-primary font-medium">
                    {String(i + 1).padStart(2, "0")}
                  </span>
                  <span>{step}</span>
                </div>
              ))}
            </CardContent>
          </Card>
          <p className="text-xs text-muted-foreground">
            Note: losing the key means losing access — by design.
          </p>
        </section>

        <Separator className="my-10" />

        {/* ── Quickstart ── */}
        <section id="quickstart" className="space-y-4">
          <p className="text-xs text-muted-foreground uppercase tracking-widest">
            Quickstart
          </p>
          <p className="text-sm text-muted-foreground">
            Self-host in minutes. No managed service required.
          </p>
          <Card size="sm" className="bg-muted/30">
            <CardContent>
              <pre className="text-xs text-muted-foreground leading-relaxed overflow-x-auto">
{`# docker compose
curl -fsSL https://get.skjul.dev/compose.yml -o compose.yml
docker compose up -d

# open
http://localhost:8080`}
              </pre>
            </CardContent>
          </Card>
        </section>

        <Separator className="my-10" />

        {/* ── Final CTA ── */}
        <section className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="space-y-1">
            <p className="text-sm font-medium">Ready to paste without trust?</p>
            <p className="text-xs text-muted-foreground">
              Deploy it yourself — or use the hosted instance.
            </p>
          </div>
          <div className="flex gap-2">
            <Button render={<Link to="/" />}>
              Get started
              <HugeiconsIcon icon={ArrowRight01Icon} size={14} />
            </Button>
            <Button variant="outline" render={<a href="#quickstart" />}>
              Read the docs
            </Button>
          </div>
        </section>

        <Separator className="my-10" />

        {/* ── Footer ── */}
        <footer className="text-xs text-muted-foreground space-y-2">
          <p>© {new Date().getFullYear()} skjul · E2E encrypted pastebin · MIT</p>
          <div className="flex gap-3">
            <a
              href="https://github.com/skjul/skjul"
              target="_blank"
              rel="noreferrer"
              className="underline underline-offset-4 hover:text-foreground"
            >
              GitHub
            </a>
            <a
              href="#quickstart"
              className="underline underline-offset-4 hover:text-foreground"
            >
              Docs
            </a>
            <a
              href="#"
              className="underline underline-offset-4 hover:text-foreground"
            >
              Security
            </a>
          </div>
        </footer>
      </div>
    </div>
  );
}
