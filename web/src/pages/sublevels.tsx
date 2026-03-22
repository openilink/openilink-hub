import { useEffect, useState } from "react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import { Plus, Trash2, RotateCw, Copy, Check } from "lucide-react";
import { api } from "../lib/api";

export function SublevelsPage() {
  const [subs, setSubs] = useState<any[]>([]);
  const [bots, setBots] = useState<any[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newBotID, setNewBotID] = useState("");

  async function load() {
    const [s, b] = await Promise.all([api.listSublevels(), api.listBots()]);
    setSubs(s || []);
    setBots(b || []);
    if (b?.length && !newBotID) setNewBotID(b[0].id);
  }

  useEffect(() => { load(); }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!newName || !newBotID) return;
    await api.createSublevel(newBotID, newName);
    setNewName("");
    setShowCreate(false);
    load();
  }

  async function handleDelete(id: string) {
    if (!confirm("确定删除此分发通道？")) return;
    await api.deleteSublevel(id);
    load();
  }

  async function handleRotate(id: string) {
    if (!confirm("重新生成 API Key？旧 Key 将立即失效。")) return;
    await api.rotateKey(id);
    load();
  }

  const botMap = Object.fromEntries(bots.map((b) => [b.id, b]));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">分发通道</h2>
        <Button onClick={() => setShowCreate(true)} size="sm" disabled={bots.length === 0}>
          <Plus className="w-4 h-4 mr-1" /> 新建通道
        </Button>
      </div>

      {showCreate && (
        <Card>
          <form onSubmit={handleCreate} className="flex gap-3 items-end">
            <div className="flex-1 space-y-1">
              <label className="text-xs text-[var(--muted-foreground)]">通道名称</label>
              <Input value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="例：客服系统" autoFocus />
            </div>
            <div className="space-y-1">
              <label className="text-xs text-[var(--muted-foreground)]">关联 Bot</label>
              <select
                className="w-48 rounded-[var(--radius)] border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-sm"
                value={newBotID}
                onChange={(e) => setNewBotID(e.target.value)}
              >
                {bots.map((b) => <option key={b.id} value={b.id}>{b.name || b.bot_id}</option>)}
              </select>
            </div>
            <Button type="submit" size="sm">创建</Button>
            <Button type="button" variant="ghost" size="sm" onClick={() => setShowCreate(false)}>取消</Button>
          </form>
        </Card>
      )}

      <div className="grid gap-3">
        {subs.map((sub) => (
          <SublevelItem
            key={sub.id}
            sub={sub}
            botName={botMap[sub.bot_db_id]?.name || sub.bot_db_id}
            onDelete={handleDelete}
            onRotate={handleRotate}
          />
        ))}
        {subs.length === 0 && (
          <p className="text-center text-sm text-[var(--muted-foreground)] py-12">
            {bots.length === 0 ? "请先绑定一个 Bot" : "还没有分发通道，点击上方按钮创建"}
          </p>
        )}
      </div>
    </div>
  );
}

function SublevelItem({ sub, botName, onDelete, onRotate }: {
  sub: any; botName: string; onDelete: (id: string) => void; onRotate: (id: string) => void;
}) {
  const [copied, setCopied] = useState(false);
  const wsUrl = `ws://${location.host}/api/ws?key=${sub.api_key}`;

  function copyKey() {
    navigator.clipboard.writeText(sub.api_key);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <Card className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <p className="font-medium">{sub.name}</p>
          <p className="text-xs text-[var(--muted-foreground)]">
            Bot: {botName}
            {sub.enabled ? "" : " (已禁用)"}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant={sub.enabled ? "default" : "outline"}>
            {sub.enabled ? "启用" : "禁用"}
          </Badge>
          <Button variant="ghost" size="sm" onClick={() => onRotate(sub.id)} title="重新生成 Key">
            <RotateCw className="w-4 h-4" />
          </Button>
          <Button variant="ghost" size="sm" onClick={() => onDelete(sub.id)}>
            <Trash2 className="w-4 h-4 text-[var(--destructive)]" />
          </Button>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <code className="flex-1 text-xs bg-[var(--background)] border border-[var(--border)] rounded px-3 py-2 font-mono truncate">
          {sub.api_key}
        </code>
        <Button variant="outline" size="sm" onClick={copyKey}>
          {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
        </Button>
      </div>

      <p className="text-xs text-[var(--muted-foreground)] font-mono truncate">
        WebSocket: {wsUrl}
      </p>
    </Card>
  );
}
