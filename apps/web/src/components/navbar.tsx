import { Link } from "@tanstack/react-router";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  Github01Icon,
  Menu01Icon,
  PlusSignIcon,
} from "@hugeicons/core-free-icons";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { ModeToggle } from "@/components/mode-toggle";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useAuth } from "@/lib/auth";

export function Navbar() {
  const { user, isLoading, logout } = useAuth();

  const navItems = (
    <>
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
          <Button size="sm" render={<Link to="/new" />}>
            <HugeiconsIcon icon={PlusSignIcon} size={14} />
            New paste
          </Button>
          <Button
            variant="outline"
            size="sm"
            render={<Link to="/dashboard" search={{ tab: "account" }} />}
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
            href="https://github.com/bmardale/skjul"
            target="_blank"
            rel="noreferrer"
          />
        }
      >
        <HugeiconsIcon icon={Github01Icon} size={14} />
      </Button>
      <ModeToggle />
    </>
  );

  const mobileNavItems = (
    <>
      {isLoading ? (
        <div
          aria-hidden="true"
          className="h-7 w-28 border border-border bg-muted/50 animate-pulse px-2 py-2"
        />
      ) : user !== null ? (
        <>
          <div className="px-2 py-2 font-mono text-xs text-muted-foreground">
            {user.username}
          </div>
          <DropdownMenuSeparator />
          <DropdownMenuItem render={<Link to="/new" />}>
            <HugeiconsIcon icon={PlusSignIcon} size={14} />
            New paste
          </DropdownMenuItem>
          <DropdownMenuItem
            render={<Link to="/dashboard" search={{ tab: "account" }} />}
          >
            Dashboard
          </DropdownMenuItem>
          <DropdownMenuItem onClick={logout}>Logout</DropdownMenuItem>
        </>
      ) : (
        <>
          <DropdownMenuItem render={<Link to="/login" />}>
            Login
          </DropdownMenuItem>
          <DropdownMenuItem render={<Link to="/register" />}>
            Register
          </DropdownMenuItem>
        </>
      )}

      <DropdownMenuSeparator />
      <DropdownMenuItem
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
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <div className="px-2 py-2">
        <ModeToggle />
      </div>
    </>
  );

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

        <div className="hidden md:flex md:items-center md:gap-1">
          {navItems}
        </div>

        <div className="flex md:hidden">
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button variant="ghost" size="icon-sm" aria-label="Open menu" />
              }
            >
              <HugeiconsIcon icon={Menu01Icon} size={20} />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="min-w-40">
              {mobileNavItems}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
      <Separator />
    </nav>
  );
}
