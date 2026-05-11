import { jsx as _jsx } from "react/jsx-runtime";
import { createContext, useContext, useEffect, useRef, useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "./query-keys";
// Event types matching backend push.Event* constants.
const EventTraceCompleted = "trace_completed";
const EventMessageNew = "message_new";
const EventWebhookLog = "webhook_log";
const EventBotStatus = "bot_status";
/** Manages a single reconnecting WebSocket to /api/ws. */
class PushClient {
  constructor() {
    this.ws = null;
    this.subs = new Map(); // botID -> refcount
    this.listeners = new Set();
    this.reconnectTimer = null;
    this.reconnectDelay = 1000;
    this.closed = false;
    this.everConnected = false;
    this.failCount = 0;
  }
  connect() {
    if (this.closed) return;
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    const url = `${proto}//${location.host}/api/ws`;
    const ws = new WebSocket(url);
    ws.onopen = () => {
      this.everConnected = true;
      this.failCount = 0;
      this.reconnectDelay = 1000;
      // Re-subscribe to all active subscriptions.
      const botIDs = [...this.subs.keys()];
      if (botIDs.length > 0) {
        this.trySend({ type: "subscribe", data: { bot_ids: botIDs } });
      }
    };
    ws.onmessage = (e) => {
      try {
        const env = JSON.parse(e.data);
        this.listeners.forEach((fn) => fn(env));
      } catch {
        /* ignore malformed */
      }
    };
    ws.onclose = () => {
      this.ws = null;
      if (this.closed) return;
      if (this.everConnected) {
        // Was connected before — always reconnect (server restart, etc.)
        this.scheduleReconnect();
      } else if (++this.failCount < 3) {
        // Never connected — retry a few times for transient failures.
        this.scheduleReconnect();
      }
      // After 3 consecutive pre-open failures, stop (likely 401/auth).
    };
    ws.onerror = () => {
      ws.close();
    };
    this.ws = ws;
  }
  scheduleReconnect() {
    if (this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000);
      this.connect();
    }, this.reconnectDelay);
  }
  subscribe(botID) {
    const prev = this.subs.get(botID) ?? 0;
    this.subs.set(botID, prev + 1);
    if (prev === 0) this.trySend({ type: "subscribe", data: { bot_ids: [botID] } });
  }
  unsubscribe(botID) {
    const cur = (this.subs.get(botID) ?? 0) - 1;
    if (cur <= 0) {
      this.subs.delete(botID);
      this.trySend({ type: "unsubscribe", data: { bot_ids: [botID] } });
    } else {
      this.subs.set(botID, cur);
    }
  }
  trySend(msg) {
    try {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify(msg));
      }
    } catch {
      /* socket closed between check and send */
    }
  }
  addListener(fn) {
    this.listeners.add(fn);
  }
  removeListener(fn) {
    this.listeners.delete(fn);
  }
  close() {
    this.closed = true;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.ws?.close();
  }
}
const PushContext = createContext(null);
export function PushProvider({ children }) {
  const [client, setClient] = useState(null);
  const qc = useQueryClient();
  useEffect(() => {
    // Create a fresh client each mount to handle React StrictMode double-mount.
    const c = new PushClient();
    setClient(c);
    c.connect();
    // Global listener that invalidates React Query caches.
    const handler = (env) => {
      const botID = env.data?.bot_id;
      if (!botID) return;
      switch (env.type) {
        case EventTraceCompleted:
          qc.invalidateQueries({ queryKey: ["bots", botID, "traces"] });
          break;
        case EventMessageNew:
          qc.invalidateQueries({ queryKey: ["bots", botID, "messages"] });
          // Webhook logs are also generated during message processing.
          qc.invalidateQueries({ queryKey: ["bots", botID, "webhook-logs"] });
          break;
        case EventWebhookLog:
          qc.invalidateQueries({ queryKey: ["bots", botID, "webhook-logs"] });
          break;
        case EventBotStatus:
          qc.invalidateQueries({ queryKey: queryKeys.bots.all() });
          break;
      }
    };
    c.addListener(handler);
    return () => {
      c.removeListener(handler);
      c.close();
      setClient(null);
    };
  }, [qc]);
  return _jsx(PushContext.Provider, { value: client, children: children });
}
/** Subscribe to push events for a bot. Automatically manages ref counting. */
export function useBotPush(botID) {
  const client = useContext(PushContext);
  useEffect(() => {
    if (!client || !botID) return;
    client.subscribe(botID);
    return () => client.unsubscribe(botID);
  }, [client, botID]);
}
/** Listen to raw push events. */
export function usePushListener(fn) {
  const client = useContext(PushContext);
  const fnRef = useRef(fn);
  fnRef.current = fn;
  const stable = useCallback((env) => fnRef.current(env), []);
  useEffect(() => {
    if (!client) return;
    client.addListener(stable);
    return () => client.removeListener(stable);
  }, [client, stable]);
}
