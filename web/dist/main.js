import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { Progress, ProgressProvider, useAnchorProgress } from "@bprogress/react";
import { queryClient } from "./lib/query-client";
import "./index.css";
import { HomePage } from "./pages/home";
import { LoginPage } from "./pages/login";
import { Layout } from "./components/layout";
import { BotsPage } from "./pages/bots";
import { BotDetailPage } from "./pages/bot-detail";
import { SettingsPage } from "./pages/settings";
import { AdminOverviewPage } from "./pages/admin-overview";
import { AdminUsersPage } from "./pages/admin-users";
import { AdminReviewsPage } from "./pages/admin-reviews";
import { AppsPage } from "./pages/apps";
import { AppDetailPage } from "./pages/app-detail";
import { DashboardOverviewPage } from "./pages/dashboard-overview";
import { TracesPage } from "./pages/traces";
import { TraceDetailPage } from "./pages/trace-detail";
import { ConsolePage } from "./pages/console/console-page";
import { ThemeProvider } from "./lib/theme";
import { PushProvider } from "./lib/ws";
import { PostHogIdentify } from "./lib/posthog";
import { TooltipProvider } from "./components/ui/tooltip";
import { Toaster } from "./components/ui/toaster";
import { OnboardingPage } from "./pages/onboarding";
import { InstallationDetailPage } from "./pages/installation-detail";
import { InstallAppPage } from "./pages/install-app";
import { DeveloperAppsPage } from "./pages/developer-apps";
function RouterProgress() {
  useAnchorProgress({ startOnLoad: false });
  return null;
}
createRoot(document.getElementById("root")).render(
  _jsx(StrictMode, {
    children: _jsx(QueryClientProvider, {
      client: queryClient,
      children: _jsx(PushProvider, {
        children: _jsx(ThemeProvider, {
          children: _jsx(TooltipProvider, {
            children: _jsxs(ProgressProvider, {
              color: "oklch(0.693 0.195 151.5)",
              children: [
                _jsxs(BrowserRouter, {
                  children: [
                    _jsx(RouterProgress, {}),
                    _jsx(Progress, {}),
                    _jsx(PostHogIdentify, {}),
                    _jsxs(Routes, {
                      children: [
                        _jsx(Route, { path: "/", element: _jsx(HomePage, {}) }),
                        _jsx(Route, { path: "/login", element: _jsx(LoginPage, {}) }),
                        _jsxs(Route, {
                          path: "/dashboard",
                          element: _jsx(Layout, {}),
                          children: [
                            _jsx(Route, {
                              index: true,
                              element: _jsx(Navigate, { to: "overview", replace: true }),
                            }),
                            _jsx(Route, {
                              path: "overview",
                              element: _jsx(DashboardOverviewPage, {}),
                            }),
                            _jsx(Route, { path: "onboarding", element: _jsx(OnboardingPage, {}) }),
                            _jsx(Route, { path: "accounts", element: _jsx(BotsPage, {}) }),
                            _jsx(Route, { path: "accounts/:id", element: _jsx(BotDetailPage, {}) }),
                            _jsx(Route, {
                              path: "accounts/:id/apps/:iid",
                              element: _jsx(InstallationDetailPage, {}),
                            }),
                            _jsx(Route, {
                              path: "accounts/:id/install/:appId",
                              element: _jsx(InstallAppPage, {}),
                            }),
                            _jsx(Route, {
                              path: "accounts/:id/traces",
                              element: _jsx(TracesPage, {}),
                            }),
                            _jsx(Route, {
                              path: "accounts/:id/traces/:traceId",
                              element: _jsx(TraceDetailPage, {}),
                            }),
                            _jsx(Route, {
                              path: "accounts/:id/console",
                              element: _jsx(ConsolePage, {}),
                            }),
                            _jsx(Route, { path: "apps", element: _jsx(AppsPage, {}) }),
                            _jsx(Route, { path: "apps/:id", element: _jsx(AppDetailPage, {}) }),
                            _jsx(Route, {
                              path: "developer/apps",
                              element: _jsx(DeveloperAppsPage, {}),
                            }),
                            _jsx(Route, {
                              path: "developer/apps/:id",
                              element: _jsx(AppDetailPage, {}),
                            }),
                            _jsx(Route, {
                              path: "admin",
                              element: _jsx(Navigate, {
                                to: "/dashboard/admin/overview",
                                replace: true,
                              }),
                            }),
                            _jsx(Route, {
                              path: "admin/overview",
                              element: _jsx(AdminOverviewPage, {}),
                            }),
                            _jsx(Route, { path: "admin/users", element: _jsx(AdminUsersPage, {}) }),
                            _jsx(Route, {
                              path: "admin/reviews",
                              element: _jsx(AdminReviewsPage, {}),
                            }),
                            _jsxs(Route, {
                              path: "settings",
                              element: _jsx(SettingsPage, {}),
                              children: [
                                _jsx(Route, {
                                  index: true,
                                  element: _jsx(Navigate, { to: "profile", replace: true }),
                                }),
                                _jsx(Route, { path: "profile", element: null }),
                                _jsx(Route, { path: "security", element: null }),
                              ],
                            }),
                          ],
                        }),
                      ],
                    }),
                  ],
                }),
                _jsx(Toaster, {}),
              ],
            }),
          }),
        }),
      }),
    }),
  }),
);
