import { ThemeProvider } from "@/components/theme-provider";
import { Navbar } from "@/components/navbar";
import { createRootRoute, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";

const RootLayout = () => (
  <ThemeProvider defaultTheme="system" storageKey="skjul-ui-theme">
    <div className="min-h-svh bg-background text-foreground">
      <Navbar />
      <Outlet />
    </div>
    <TanStackRouterDevtools />
  </ThemeProvider>
);

export const Route = createRootRoute({ component: RootLayout });
