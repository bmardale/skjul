import { ThemeProvider } from "@/components/theme-provider";
import { AuthProvider } from "@/lib/auth";
import { Navbar } from "@/components/navbar";
import { createRootRoute, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";

const RootLayout = () => (
  <ThemeProvider defaultTheme="system" storageKey="skjul-ui-theme">
    <AuthProvider>
      <div className="min-h-svh bg-background text-foreground">
        <Navbar />
        <Outlet />
      </div>
    </AuthProvider>
    <TanStackRouterDevtools />
  </ThemeProvider>
);

export const Route = createRootRoute({ component: RootLayout });
