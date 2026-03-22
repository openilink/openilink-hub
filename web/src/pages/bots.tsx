import { useEffect, useState } from "react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Plus, Trash2, RefreshCw, QrCode } from "lucide-react";
import { api } from "../lib/api";

const statusColor: Record<string, "default" | "destructive" | "outline"> = {
  connected: "default",
  disconnected: "outline",
  error: "destructive",
  session_expired: "destructive",
};

export function BotsPage() {
  const [bots, setBots] = useState<any[]>([]);
  const [binding, setBinding] = useState(false);
  const [qrUrl, setQrUrl] = useState("");
  const [bindStatus, setBindStatus] = useState("");

  async function loadBots() {
    try {
      const data = await api.listBots();
      setBots(data || []);
    } catch {}
  }

  useEffect(() => { loadBots(); }, []);

  async function startBind() {
    setBinding(true);
    setBindStatus("获取二维码...");
    try {
      const { session_id, qr_url } = await api.bindStart();
      setQrUrl(qr_url);
      setBindStatus("请用微信扫描二维码");

      // SSE for status
      const es = new EventSource(`/api/bots/bind/status/${session_id}`);
      es.addEventListener("status", (e) => {
        const data = JSON.parse(e.data);
        switch (data.status) {
          case "wait": break;
          case "scanned": setBindStatus("已扫码，请在微信上确认..."); break;
          case "refreshed": setQrUrl(data.qr_url); setBindStatus("二维码已刷新，请重新扫描"); break;
          case "connected":
            setBindStatus("绑定成功！");
            es.close();
            setTimeout(() => { setBinding(false); setQrUrl(""); loadBots(); }, 1500);
            break;
        }
      });
      es.addEventListener("error", () => {
        setBindStatus("绑定失败");
        es.close();
      });
    } catch (err: any) {
      setBindStatus("失败: " + err.message);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("确定删除此 Bot？关联的分发通道也会被删除。")) return;
    await api.deleteBot(id);
    loadBots();
  }

  async function handleReconnect(id: string) {
    await api.reconnectBot(id);
    loadBots();
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Bot 管理</h2>
        <Button onClick={startBind} disabled={binding} size="sm">
          <Plus className="w-4 h-4 mr-1" /> 绑定新 Bot
        </Button>
      </div>

      {binding && (
        <Card className="flex flex-col items-center gap-4 py-8">
          <QrCode className="w-6 h-6 text-[var(--muted-foreground)]" />
          {qrUrl && <img src={qrUrl} alt="QR Code" className="w-56 h-56 rounded-lg bg-white p-2" />}
          <p className="text-sm text-[var(--muted-foreground)]">{bindStatus}</p>
          <Button variant="ghost" size="sm" onClick={() => { setBinding(false); setQrUrl(""); }}>取消</Button>
        </Card>
      )}

      <div className="grid gap-3">
        {bots.map((bot) => (
          <BotItem key={bot.id} bot={bot} onDelete={handleDelete} onReconnect={handleReconnect} onRename={loadBots} />
        ))}
        {bots.length === 0 && !binding && (
          <p className="text-center text-sm text-[var(--muted-foreground)] py-12">
            还没有 Bot，点击上方按钮绑定一个
          </p>
        )}
      </div>
    </div>
  );
}

function BotItem({ bot, onDelete, onReconnect, onRename }: {
  bot: any; onDelete: (id: string) => void; onReconnect: (id: string) => void; onRename: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(bot.name);

  async function saveName() {
    await api.renameBot(bot.id, name);
    setEditing(false);
    onRename();
  }

  return (
    <Card className="flex items-center justify-between">
      <div className="space-y-1">
        {editing ? (
          <div className="flex gap-2 items-center">
            <Input value={name} onChange={(e) => setName(e.target.value)} className="w-48" />
            <Button size="sm" onClick={saveName}>保存</Button>
            <Button size="sm" variant="ghost" onClick={() => setEditing(false)}>取消</Button>
          </div>
        ) : (
          <p className="font-medium cursor-pointer hover:text-[var(--primary)]" onClick={() => setEditing(true)}>
            {bot.name || bot.bot_id}
          </p>
        )}
        <p className="text-xs text-[var(--muted-foreground)] font-mono">{bot.bot_id}</p>
      </div>
      <div className="flex items-center gap-3">
        <Badge variant={statusColor[bot.status] || "outline"}>{bot.status}</Badge>
        {bot.status !== "connected" && (
          <Button variant="ghost" size="sm" onClick={() => onReconnect(bot.id)}>
            <RefreshCw className="w-4 h-4" />
          </Button>
        )}
        <Button variant="ghost" size="sm" onClick={() => onDelete(bot.id)}>
          <Trash2 className="w-4 h-4 text-[var(--destructive)]" />
        </Button>
      </div>
    </Card>
  );
}
