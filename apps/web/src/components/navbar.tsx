import { Link } from "@tanstack/react-router";
import { HugeiconsIcon } from "@hugeicons/react";
import { Github01Icon, PlusSignIcon } from "@hugeicons/core-free-icons";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { ModeToggle } from "@/components/mode-toggle";
import { useAuth } from "@/lib/auth";

export function Navbar() {
  const { user, isLoading, logout } = useAuth();

  return (
    <nav className="mx-auto w-full max-w-2xl px-4">
      <div className="flex h-12 items-center justify-between">
        <div className="flex items-center gap-2">
          <Link to="/" className="flex items-center gap-2">
            <span className="text-muted-foreground">$</span>
            <span className="text-sm font-medium tracking-tight">skjul</span>
          </Link>
          <Badge variant="outline">E2E</Badge>
        </div>

        <div className="flex items-center gap-1">
          {isLoading ? (
            <div
              aria-hidden="true"
              className="h-7 w-28 border border-border bg-muted/50 animate-pulse"
            />
          ) : user !== null ? (
            <>
              <span className="px-1 font-mono text-xs text-muted-foreground">
                {user.username}
              </span>
              <Separator orientation="vertical" className="mx-1 h-4" />
              <Button
                size="sm"
                render={<Link to="/new" />}
              >
                <HugeiconsIcon icon={PlusSignIcon} size={14} />
                New paste
              </Button>
              <Button
                variant="outline"
                size="sm"
                render={<Link to="/dashboard" />}
              >
                Dashboard
              </Button>
              <Button variant="ghost" size="xs" onClick={logout}>
                Logout
              </Button>
            </>
          ) : (
            <>
              <Button variant="ghost" size="sm" render={<Link to="/login" />}>
                Login
              </Button>
              <Button size="sm" render={<Link to="/register" />}>
                Register
              </Button>
            </>
          )}

          <Separator orientation="vertical" className="mx-1 h-4" />

          <Button
            variant="ghost"
            size="icon-sm"
            render={
              <a
                href="https://github.com/skjul/skjul"
                target="_blank"
                rel="noreferrer"
              />
            }
          >
            <HugeiconsIcon icon={Github01Icon} size={14} />
          </Button>
          <ModeToggle />
        </div>
      </div>
      <Separator />
    </nav>
  );
}
