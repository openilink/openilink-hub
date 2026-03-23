import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import QRCode from "qrcode";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Plus, Trash2, RefreshCw, Bot } from "lucide-react";
import { api } from "../lib/api";

const statusVariant: Record<string, "default" | "destructive" | "outline"> = {
  connected: "default", disconnected: "outline", error: "destructive", session_expired: "destructive",
};

export function BotsPage() {
  const [bots, setBots] = useState<any[]>([]);
  const [channels, setChannels] = useState<any[]>([]);
  const [binding, setBinding] = useState(false);
  const [qrUrl, setQrUrl] = useState("");
  const [bindStatus, setBindStatus] = useState("");
  const [hasGlobalAI, setHasGlobalAI] = useState(false);
  const [enableAI, setEnableAI] = useState(false);

  async function load() {
    const b = await api.listBots();
    setBots(b || []);
    // Load channels for all bots
    const allChannels: any[] = [];
    for (const bot of (b || [])) {
      const chs = await api.listChannels(bot.id);
      allChannels.push(...(chs || []));
    }
    setChannels(allChannels);
  }

  useEffect(() => {
    load();
    api.info().then((f) => { setHasGlobalAI(f.ai); setEnableAI(f.ai); }).catch(() => {});
  }, []);

  async function startBind() {
    setBinding(true);
    setBindStatus("获取二维码...");
    try {
      const { session_id, qr_url } = await api.bindStart();
      setQrUrl(qr_url);
      setBindStatus("请用微信扫描二维码");
      const es = new EventSource(`/api/bots/bind/status/${session_id}${enableAI ? "?enable_ai=true" : ""}`);
      es.addEventListener("status", (e) => {
        const data = JSON.parse(e.data);
        if (data.status === "scanned") setBindStatus("已扫码，请在微信确认...");
        if (data.status === "refreshed") { setQrUrl(data.qr_url); setBindStatus("二维码已刷新"); }
        if (data.status === "connected") {
          setBindStatus("绑定成功！");
          es.close();
          setTimeout(() => { setBinding(false); setQrUrl(""); load(); }, 1000);
        }
      });
      es.addEventListener("error", () => { setBindStatus("绑定失败"); es.close(); });
    } catch (err: any) {
      setBindStatus("失败: " + err.message);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold">Bot 管理</h1>
          <p className="text-xs text-muted-foreground mt-0.5">管理你的微信 Bot，扫码绑定新 Bot</p>
        </div>
        {!binding && (
          <Button onClick={startBind} variant="outline" size="sm">
            <Plus className="w-4 h-4 mr-1" /> 绑定新 Bot
          </Button>
        )}
      </div>

      {binding ? (
        <Card className="flex flex-col items-center gap-4 py-8">
          <QrCanvas url={qrUrl} />
          <p className="text-sm text-muted-foreground">{bindStatus}</p>
          {hasGlobalAI && (
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={enableAI} onChange={(e) => setEnableAI(e.target.checked)} className="w-3.5 h-3.5 accent-primary" />
              <Bot className="w-3.5 h-3.5 text-muted-foreground" />
              <span className="text-xs text-muted-foreground">自动开启内置 AI 回复</span>
            </label>
          )}
          <Button variant="ghost" size="sm" onClick={() => { setBinding(false); setQrUrl(""); }}>取消</Button>
        </Card>
      ) : null}

      {bots.map((bot) => (
        <BotCard
          key={bot.id}
          bot={bot}
          channelCount={(channels || []).filter((c) => c.bot_id === bot.id).length}
          onRefresh={load}
        />
      ))}

      {bots.length === 0 && !binding && (
        <p className="text-center text-sm text-muted-foreground py-8">点击上方按钮绑定你的第一个 Bot</p>
      )}
    </div>
  );
}

function QrCanvas({ url }: { url: string }) {
  const ref = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    if (!url || !ref.current) return;
    QRCode.toCanvas(ref.current, url, { width: 224, margin: 2, color: { dark: "#000", light: "#fff" } });
  }, [url]);
  if (!url) return null;
  return <canvas ref={ref} className="rounded-lg" />;
}

function BotCard({ bot, channelCount, onRefresh }: { bot: any; channelCount: number; onRefresh: () => void }) {
  const navigate = useNavigate();

  async function handleDelete(e: React.MouseEvent) {
    e.stopPropagation();
    if (!confirm("删除此 Bot 及其所有通道？")) return;
    await api.deleteBot(bot.id);
    onRefresh();
  }

  async function handleReconnect(e: React.MouseEvent) {
    e.stopPropagation();
    await api.reconnectBot(bot.id);
    onRefresh();
  }

  return (
    <Card
      className="flex items-center justify-between cursor-pointer hover:border-primary/50 transition-colors"
      onClick={() => navigate(`/bot/${bot.id}`)}
    >
      <div>
        <p className="font-medium text-sm">{bot.name}</p>
        <p className="text-xs text-muted-foreground font-mono mt-0.5">{bot.extra?.bot_id}</p>
        <div className="flex items-center gap-2 mt-1">
          <span className="text-xs text-muted-foreground">{channelCount} 个通道</span>
          {bot.status === "session_expired" && (
            <span className="text-[10px] text-destructive">会话过期，请重新扫码绑定</span>
          )}
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Badge variant={statusVariant[bot.status] || "outline"}>
          {bot.status === "session_expired" ? "已过期" : bot.status}
        </Badge>
        {bot.status !== "connected" && bot.status !== "session_expired" && (
          <Button variant="ghost" size="sm" onClick={handleReconnect}>
            <RefreshCw className="w-3.5 h-3.5" />
          </Button>
        )}
        <Button variant="ghost" size="sm" onClick={handleDelete}>
          <Trash2 className="w-3.5 h-3.5 text-destructive" />
        </Button>
      </div>
    </Card>
  );
}
