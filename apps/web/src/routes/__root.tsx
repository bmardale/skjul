import { ThemeProvider } from "@/components/theme-provider";
import { AuthProvider } from "@/lib/auth";
import { AppConfigProvider } from "@/lib/app-config";
import { Navbar } from "@/components/navbar";
import { createRootRoute, HeadContent, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";

const RootLayout = () => (
  <>
      <HeadContent />
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
  </>
);

export const Route = createRootRoute({ 
  head: () => ({
    meta: [
      {
        name: "description",
        content: "End-to-end encrypted pastes. Self-hostable pastebin with accounts — the server never sees plaintext.",
      },
      {
        title: "skjul - home",
      }
    ]
  }),
  component: RootLayout 
});
