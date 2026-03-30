import { createContext, useContext, useEffect, useRef, useCallback, type ReactNode } from "react";
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

    ws.onopen = () => {
      this.reconnectDelay = 1000;
      // Re-subscribe to all active subscriptions.
      const botIDs = [...this.subs.keys()];
      if (botIDs.length > 0) {
        ws.send(JSON.stringify({ type: "subscribe", data: { bot_ids: botIDs } }));
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
      if (!this.closed) this.scheduleReconnect();
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
    if (prev === 0 && this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "subscribe", data: { bot_ids: [botID] } }));
    }
  }

  unsubscribe(botID: string) {
    const cur = (this.subs.get(botID) ?? 0) - 1;
    if (cur <= 0) {
      this.subs.delete(botID);
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: "unsubscribe", data: { bot_ids: [botID] } }));
      }
    } else {
      this.subs.set(botID, cur);
    }
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
  const clientRef = useRef<PushClient | null>(null);
  const qc = useQueryClient();

  if (!clientRef.current) {
    clientRef.current = new PushClient();
  }

  useEffect(() => {
    const client = clientRef.current!;
    client.connect();

    // Global listener that invalidates React Query caches.
    const handler: Listener = (env) => {
      const botID = env.data?.bot_id;
      if (!botID) return;

      switch (env.type) {
        case EventTraceCompleted:
          qc.invalidateQueries({ queryKey: queryKeys.bots.traces(botID) });
          break;
        case EventMessageNew:
          qc.invalidateQueries({ queryKey: queryKeys.bots.messages(botID) });
          break;
        case EventWebhookLog:
          qc.invalidateQueries({ queryKey: ["bots", botID, "webhook-logs"] });
          break;
        case EventBotStatus:
          qc.invalidateQueries({ queryKey: queryKeys.bots.all() });
          break;
      }
    };
    client.addListener(handler);

    return () => {
      client.removeListener(handler);
      client.close();
    };
  }, [qc]);

  return (
    <PushContext.Provider value={clientRef.current}>
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
