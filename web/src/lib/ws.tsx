import { createContext, useContext, useEffect, useRef, useState, useCallback, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "./query-keys";

// Event types matching backend push.Event* constants.
const EventTraceCompleted = "trace_completed";
const EventMessageNew = "message_new";
const EventWebhookLog = "webhook_log";
const EventBotStatus = "bot_status";

interface PushEnvelope {
  type: string;
  data?: { bot_id?: string; trace_id?: string };
}

type Listener = (env: PushEnvelope) => void;

/** Manages a single reconnecting WebSocket to /api/ws. */
class PushClient {
  private ws: WebSocket | null = null;
  private subs = new Map<string, number>(); // botID -> refcount
  private listeners = new Set<Listener>();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectDelay = 1000;
  private closed = false;

  connect() {
    if (this.closed) return;
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    const url = `${proto}//${location.host}/api/ws`;
    const ws = new WebSocket(url);
    let opened = false;

    ws.onopen = () => {
      opened = true;
      this.reconnectDelay = 1000;
      // Re-subscribe to all active subscriptions.
      const botIDs = [...this.subs.keys()];
      if (botIDs.length > 0) {
        this.trySend({ type: "subscribe", data: { bot_ids: botIDs } });
      }
    };

    ws.onmessage = (e) => {
      try {
        const env: PushEnvelope = JSON.parse(e.data);
        this.listeners.forEach((fn) => fn(env));
      } catch { /* ignore malformed */ }
    };

    ws.onclose = () => {
      this.ws = null;
      // Don't reconnect if the connection never opened (e.g. 401 on upgrade).
      if (!this.closed && opened) this.scheduleReconnect();
    };

    ws.onerror = () => {
      ws.close();
    };

    this.ws = ws;
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30_000);
      this.connect();
    }, this.reconnectDelay);
  }

  subscribe(botID: string) {
    const prev = this.subs.get(botID) ?? 0;
    this.subs.set(botID, prev + 1);
    if (prev === 0) this.trySend({ type: "subscribe", data: { bot_ids: [botID] } });
  }

  unsubscribe(botID: string) {
    const cur = (this.subs.get(botID) ?? 0) - 1;
    if (cur <= 0) {
      this.subs.delete(botID);
      this.trySend({ type: "unsubscribe", data: { bot_ids: [botID] } });
    } else {
      this.subs.set(botID, cur);
    }
  }

  private trySend(msg: unknown) {
    try {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify(msg));
      }
    } catch { /* socket closed between check and send */ }
  }

  addListener(fn: Listener) { this.listeners.add(fn); }
  removeListener(fn: Listener) { this.listeners.delete(fn); }

  close() {
    this.closed = true;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.ws?.close();
  }
}

const PushContext = createContext<PushClient | null>(null);

export function PushProvider({ children }: { children: ReactNode }) {
  const [client, setClient] = useState<PushClient | null>(null);
  const qc = useQueryClient();

  useEffect(() => {
    // Create a fresh client each mount to handle React StrictMode double-mount.
    const c = new PushClient();
    setClient(c);
    c.connect();

    // Global listener that invalidates React Query caches.
    const handler: Listener = (env) => {
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

  return (
    <PushContext.Provider value={client}>
      {children}
    </PushContext.Provider>
  );
}

/** Subscribe to push events for a bot. Automatically manages ref counting. */
export function useBotPush(botID: string | undefined) {
  const client = useContext(PushContext);

  useEffect(() => {
    if (!client || !botID) return;
    client.subscribe(botID);
    return () => client.unsubscribe(botID);
  }, [client, botID]);
}

/** Listen to raw push events. */
export function usePushListener(fn: Listener) {
  const client = useContext(PushContext);
  const fnRef = useRef(fn);
  fnRef.current = fn;

  const stable = useCallback((env: PushEnvelope) => fnRef.current(env), []);

  useEffect(() => {
    if (!client) return;
    client.addListener(stable);
    return () => client.removeListener(stable);
  }, [client, stable]);
}
