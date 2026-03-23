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

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <BrowserRouter>
      <Routes>
        {/* Public pages */}
        <Route path="/home" element={<HomePage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/webhook-plugins" element={<PluginsPage />} />
        {/* Authenticated pages */}
        <Route element={<Layout />}>
          <Route path="/" element={<BotsPage />} />
          <Route path="/bot/:id" element={<BotDetailPage />} />
          <Route path="/bot/:id/channel/:cid" element={<ChannelDetailPage />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  </StrictMode>
);
