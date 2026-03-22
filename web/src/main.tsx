import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import "./index.css";
import { LoginPage } from "./pages/login";
import { Layout } from "./components/layout";
import { DashboardPage } from "./pages/dashboard";
import { BotsPage } from "./pages/bots";
import { SublevelsPage } from "./pages/sublevels";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<Layout />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/bots" element={<BotsPage />} />
          <Route path="/sublevels" element={<SublevelsPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  </StrictMode>
);
