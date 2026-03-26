import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import "./index.css";
import { HomePage } from "./pages/home";
import { LoginPage } from "./pages/login";
import { Layout } from "./components/layout";
import { BotsPage } from "./pages/bots";
import { BotDetailPage } from "./pages/bot-detail";
import { SettingsPage } from "./pages/settings";
import { ChannelDetailPage } from "./pages/channel-detail";
import { AdminOverviewPage } from "./pages/admin-overview";
import { AdminUsersPage } from "./pages/admin-users";
import { AdminReviewsPage } from "./pages/admin-reviews";
import { AppsPage } from "./pages/apps";
import { AppDetailPage } from "./pages/app-detail";
import { DashboardOverviewPage } from "./pages/dashboard-overview";
import { TracesPage } from "./pages/traces";
import { ConsolePage } from "./pages/console/console-page";
import { ThemeProvider } from "./lib/theme";
import { TooltipProvider } from "./components/ui/tooltip";
import { Toaster } from "./components/ui/toaster";
import { OnboardingPage } from "./pages/onboarding";
import { InstallationDetailPage } from "./pages/installation-detail";
import { InstallAppPage } from "./pages/install-app";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider>
      <TooltipProvider>
        <BrowserRouter>
          <Routes>
            {/* Public Entry */}
            <Route path="/" element={<HomePage />} />
            <Route path="/login" element={<LoginPage />} />

            {/* Main Application Shell */}
            <Route path="/dashboard" element={<Layout />}>
              <Route index element={<Navigate to="overview" replace />} />

              {/* Domain 1: Workspace */}
              <Route path="overview" element={<DashboardOverviewPage />} />
              <Route path="onboarding" element={<OnboardingPage />} />
              <Route path="accounts" element={<BotsPage />} />
              <Route path="accounts/:id" element={<BotDetailPage />} />
              <Route path="accounts/:id/channels" element={<BotDetailPage />}>
                <Route index element={null} />
              </Route>
              <Route path="accounts/:id/apps/:iid" element={<InstallationDetailPage />} />
              <Route path="accounts/:id/install/:appId" element={<InstallAppPage />} />
              <Route path="accounts/:id/traces" element={<TracesPage />} />
              <Route path="accounts/:id/console" element={<ConsolePage />} />
              <Route path="accounts/:id/channel/:cid" element={<ChannelDetailPage />}>
                <Route index element={<Navigate to="overview" replace />} />
                <Route path="overview" element={null} />
                <Route path="webhook" element={null} />
                <Route path="ai" element={null} />
                <Route path="filter" element={null} />
                <Route path="logs" element={null} />
              </Route>

              {/* Apps */}
              <Route path="apps" element={<AppsPage />}>
                <Route index element={<Navigate to="my" replace />} />
                <Route path="my" element={null} />
                <Route path="marketplace" element={null} />
              </Route>
              <Route path="apps/:id" element={<AppDetailPage />} />

              {/* Domain 4: Management & Ops */}
              <Route path="admin" element={<Navigate to="/dashboard/admin/overview" replace />} />
              <Route path="admin/overview" element={<AdminOverviewPage />} />
              <Route path="admin/users" element={<AdminUsersPage />} />
              <Route path="admin/reviews" element={<AdminReviewsPage />} />
              <Route path="settings" element={<SettingsPage />}>
                <Route index element={<Navigate to="profile" replace />} />
                <Route path="profile" element={null} />
                <Route path="security" element={null} />
              </Route>
            </Route>
          </Routes>
        </BrowserRouter>
        <Toaster />
      </TooltipProvider>
    </ThemeProvider>
  </StrictMode>,
);
