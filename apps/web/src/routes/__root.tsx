import { ThemeProvider } from "@/components/theme-provider";
import { AuthProvider } from "@/lib/auth";
import { AppConfigProvider } from "@/lib/app-config";
import { Navbar } from "@/components/navbar";
import { createRootRoute, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";

const RootLayout = () => (
  <ThemeProvider defaultTheme="system" storageKey="skjul-ui-theme">
    <AppConfigProvider>
    <AuthProvider>
      <div className="min-h-svh bg-background text-foreground">
        <Navbar />
        <Outlet />
      </div>
    </AuthProvider>
    </AppConfigProvider>
    {import.meta.env.DEV && <TanStackRouterDevtools />}
  </ThemeProvider>
);

export const Route = createRootRoute({ component: RootLayout });
