import { useEffect } from "react";
import { useAuthStore } from "@/stores/auth-store";
const rawToken = import.meta.env.VITE_PUBLIC_POSTHOG_PROJECT_TOKEN;
const token = rawToken?.trim() || undefined;
const rawHost = import.meta.env.VITE_PUBLIC_POSTHOG_HOST;
const host = rawHost?.trim() || "https://us.i.posthog.com";
const enabled = !!token;
let instance = null;
let pending = null;
let lastIdentifiedId = null;
function load() {
  if (!enabled) return Promise.resolve(null);
  if (instance) return Promise.resolve(instance);
  if (pending) return pending;
  pending = import("posthog-js")
    .then((mod) => {
      mod.default.init(token, {
        api_host: host,
        capture_pageview: "history_change",
        capture_pageleave: true,
        person_profiles: "identified_only",
      });
      instance = mod.default;
      pending = null;
      return instance;
    })
    .catch((err) => {
      console.warn("PostHog init failed:", err);
      pending = null;
      return null;
    });
  return pending;
}
export function PostHogIdentify() {
  const user = useAuthStore((s) => s.user);
  useEffect(() => {
    let cancelled = false;
    load().then((ph) => {
      if (!ph || cancelled) return;
      if (user) {
        if (lastIdentifiedId && lastIdentifiedId !== user.id) ph.reset();
        ph.identify(user.id, {
          username: user.username,
          display_name: user.display_name,
          role: user.role,
        });
        lastIdentifiedId = user.id;
      } else if (lastIdentifiedId) {
        ph.reset();
        lastIdentifiedId = null;
      }
    });
    return () => {
      cancelled = true;
    };
  }, [user]);
  return null;
}
