import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import "./index.css";
import { HomePage } from "./pages/home";
import { LoginPage } from "./pages/login";
import { Layout } from "./components/layout";
import { BotsPage } from "./pages/bots";
import { BotDetailPage } from "./pages/bot-detail";
import { SettingsPage } from "./pages/settings";
import { PluginsPage } from "./pages/plugins";
import { ChannelDetailPage } from "./pages/channel-detail";
import { AdminPage } from "./pages/admin";
import { PluginDebugPage } from "./pages/plugin-debug";
import { AppsPage } from "./pages/apps";
import { AppDetailPage } from "./pages/app-detail";
import { ThemeProvider } from "./lib/theme";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider>
    <BrowserRouter>
      <Routes>
        {/* Public */}
        <Route path="/" element={<HomePage />} />
        <Route path="/webhook-plugins" element={<PluginsPage />} />
        <Route path="/login" element={<LoginPage />} />
        {/* Dashboard */}
        <Route path="/dashboard" element={<Layout />}>
          <Route index element={<BotsPage />} />
          <Route path="bot/:id" element={<BotDetailPage />} />
          <Route path="bot/:id/channel/:cid" element={<ChannelDetailPage />} />
          {/* Webhook plugins */}
          <Route path="webhook-plugins" element={<PluginsPage embedded tab="marketplace" />} />
          <Route path="webhook-plugins/my" element={<PluginsPage embedded tab="my" />} />
          <Route path="webhook-plugins/debug" element={<PluginDebugPage />} />
          <Route path="webhook-plugins/review" element={<PluginsPage embedded tab="review" />} />
          {/* Apps */}
          <Route path="apps" element={<AppsPage />} />
          <Route path="apps/:id" element={<AppDetailPage />} />
          {/* Settings */}
          <Route path="settings" element={<SettingsPage />} />
          {/* Admin */}
          <Route path="admin" element={<AdminPage tab="dashboard" />} />
          <Route path="admin/users" element={<AdminPage tab="users" />} />
          <Route path="admin/config" element={<AdminPage tab="config" />} />
        </Route>
      </Routes>
    </BrowserRouter>
    </ThemeProvider>
  </StrictMode>
);
