import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Plus, Blocks, X, Download } from "lucide-react";
import { api } from "../lib/api";

function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9\u4e00-\u9fff]+/g, "-")
    .replace(/^-|-$/g, "")
    .slice(0, 32);
}

export function AppsPage() {
  const [tab, setTab] = useState<"marketplace" | "my">("marketplace");

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold">App 管理</h1>
          <p className="text-xs text-muted-foreground mt-0.5">发现和安装 App，或创建你自己的 App</p>
        </div>
      </div>

      <div className="flex border rounded-lg overflow-hidden w-fit">
        <button
          className={`px-3 py-1 text-xs cursor-pointer ${tab === "marketplace" ? "bg-secondary" : "text-muted-foreground"}`}
          onClick={() => setTab("marketplace")}
        >
          市场
        </button>
        <button
          className={`px-3 py-1 text-xs cursor-pointer ${tab === "my" ? "bg-secondary" : "text-muted-foreground"}`}
          onClick={() => setTab("my")}
        >
          我的
        </button>
      </div>

      {tab === "marketplace" && <MarketplaceTab />}
      {tab === "my" && <MyAppsTab />}
    </div>
  );
}

// ==================== Marketplace ====================

function MarketplaceTab() {
  const [apps, setApps] = useState<any[]>([]);
  const [installApp, setInstallApp] = useState<any>(null);

  useEffect(() => {
    api
      .listApps({ listed: true })
      .then((list) => setApps(list || []))
      .catch(() => {});
  }, []);

  return (
    <div className="space-y-3">
      {apps.length === 0 && (
        <p className="text-center text-sm text-muted-foreground py-8">暂无上架的 App</p>
      )}
      <div className="grid gap-3 sm:grid-cols-2">
        {apps.map((app) => (
          <Card key={app.id} className="space-y-2">
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-3">
                {app.icon ? (
                  <img
                    src={app.icon}
                    alt=""
                    className="w-10 h-10 rounded-lg object-cover shrink-0"
                  />
                ) : (
                  <div className="w-10 h-10 rounded-lg bg-secondary flex items-center justify-center shrink-0">
                    <Blocks className="w-5 h-5 text-muted-foreground" />
                  </div>
                )}
                <div className="min-w-0">
                  <p className="font-medium text-sm">{app.name}</p>
                  {app.description && (
                    <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">
                      {app.description}
                    </p>
                  )}
                </div>
              </div>
              <Button size="sm" variant="outline" onClick={() => setInstallApp(app)}>
                <Download className="w-3.5 h-3.5 mr-1" /> 安装
              </Button>
            </div>
            {app.commands?.length > 0 && (
              <div className="flex flex-wrap gap-1">
                {app.commands.map((cmd: any) => (
                  <Badge key={cmd.name || cmd} variant="outline" className="text-xs font-mono">
                    /{(typeof cmd === "string" ? cmd : cmd.name).replace(/^\//, "")}
                  </Badge>
                ))}
              </div>
            )}
            {app.scopes?.length > 0 && (
              <div className="flex flex-wrap gap-1">
                {app.scopes.map((s: string) => (
                  <span
                    key={s}
                    className="text-xs text-muted-foreground bg-secondary px-1.5 py-0.5 rounded"
                  >
                    {s}
                  </span>
                ))}
              </div>
            )}
          </Card>
        ))}
      </div>

      {installApp && <InstallModal app={installApp} onClose={() => setInstallApp(null)} />}
    </div>
  );
}

// ==================== Install Modal ====================

function InstallModal({ app, onClose }: { app: any; onClose: () => void }) {
  const [bots, setBots] = useState<any[]>([]);
  const [botId, setBotId] = useState("");
  const [handle, setHandle] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState<{ handle: string; command?: string } | null>(null);

  useEffect(() => {
    api
      .listBots()
      .then((list) => {
        const items = list || [];
        setBots(items);
        if (items.length > 0) setBotId(items[0].id);
      })
      .catch(() => {});
  }, []);

  async function handleInstall() {
    if (!botId) {
      setError("请选择一个 Bot");
      return;
    }
    setSaving(true);
    setError("");
    try {
      await api.installApp(app.id, { bot_id: botId, handle: handle.trim() || undefined });
      const firstCmd = app.commands?.[0];
      const rawName = firstCmd ? (typeof firstCmd === "string" ? firstCmd : firstCmd.name) : undefined;
      const cmdName = rawName ? "/" + rawName.replace(/^\//, "") : undefined;
      setSuccess({ handle: handle.trim(), command: cmdName });
    } catch (err: any) {
      setError(err.message);
    }
    setSaving(false);
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={onClose}
    >
      <div
        className="bg-background border rounded-xl p-5 max-w-md w-full mx-4 space-y-4"
        onClick={(e) => e.stopPropagation()}
      >
        {success ? (
          <>
            <p className="text-sm font-medium">安装成功！</p>
            <p className="text-xs text-muted-foreground">
              {success.handle && (
                <>
                  发送 <code className="bg-secondary px-1 py-0.5 rounded">@{success.handle}</code>
                </>
              )}
              {success.handle && success.command && " 或 "}
              {success.command && (
                <>
                  发送 <code className="bg-secondary px-1 py-0.5 rounded">{success.command}</code>
                </>
              )}
              {!success.handle && !success.command && "App 已安装"}
              {(success.handle || success.command) && " 测试"}
            </p>
            <div className="flex justify-end">
              <Button size="sm" onClick={onClose}>
                确认
              </Button>
            </div>
          </>
        ) : (
          <>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                {app.icon ? (
                  <img src={app.icon} alt="" className="w-8 h-8 rounded-lg object-cover" />
                ) : (
                  <div className="w-8 h-8 rounded-lg bg-secondary flex items-center justify-center">
                    <Blocks className="w-4 h-4 text-muted-foreground" />
                  </div>
                )}
                <div>
                  <p className="text-sm font-medium">安装 {app.name}</p>
                  {app.description && (
                    <p className="text-xs text-muted-foreground">{app.description}</p>
                  )}
                </div>
              </div>
              <button onClick={onClose} className="cursor-pointer">
                <X className="w-4 h-4 text-muted-foreground" />
              </button>
            </div>

            <div className="space-y-2">
              <label className="text-xs text-muted-foreground">选择 Bot</label>
              <select
                value={botId}
                onChange={(e) => setBotId(e.target.value)}
                className="w-full h-8 rounded-md border border-input bg-transparent px-3 text-xs focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:border-ring"
              >
                {bots.length === 0 && <option value="">无可用 Bot</option>}
                {bots.map((b) => (
                  <option key={b.id} value={b.id}>
                    {b.name || b.id}
                  </option>
                ))}
              </select>
            </div>

            <div className="space-y-1">
              <label className="text-xs text-muted-foreground">Handle（可选，用于 @提及）</label>
              <Input
                placeholder="留空则只能通过 /command 触发"
                value={handle}
                onChange={(e) => setHandle(e.target.value)}
                className="h-8 text-xs font-mono"
              />
            </div>

            {error && <p className="text-xs text-destructive">{error}</p>}

            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={onClose}>
                取消
              </Button>
              <Button size="sm" onClick={handleInstall} disabled={saving}>
                {saving ? "..." : "安装"}
              </Button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

// ==================== My Apps ====================

function MyAppsTab() {
  const navigate = useNavigate();
  const [apps, setApps] = useState<any[]>([]);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState({ name: "", slug: "", description: "", icon: "" });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    try {
      const list = await api.listApps();
      setApps(list || []);
    } catch {}
  }

  useEffect(() => {
    load();
  }, []);

  function handleNameChange(name: string) {
    setForm((f) => ({ ...f, name, slug: slugify(name) }));
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!form.name.trim()) {
      setError("名称不能为空");
      return;
    }
    if (!form.slug.trim()) {
      setError("Slug 不能为空");
      return;
    }
    setSaving(true);
    try {
      await api.createApp(form);
      setForm({ name: "", slug: "", description: "", icon: "" });
      setCreating(false);
      load();
    } catch (err: any) {
      setError(err.message);
    }
    setSaving(false);
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        {!creating && (
          <Button onClick={() => setCreating(true)} variant="outline" size="sm">
            <Plus className="w-4 h-4 mr-1" /> 创建 App
          </Button>
        )}
      </div>

      {creating && (
        <Card className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium">创建新 App</h3>
            <button
              onClick={() => {
                setCreating(false);
                setError("");
              }}
              className="cursor-pointer"
            >
              <X className="w-4 h-4 text-muted-foreground" />
            </button>
          </div>
          <form onSubmit={handleCreate} className="space-y-2">
            <Input
              placeholder="App 名称"
              value={form.name}
              onChange={(e) => handleNameChange(e.target.value)}
              className="h-8 text-xs"
            />
            <Input
              placeholder="Slug（URL 标识符）"
              value={form.slug}
              onChange={(e) => setForm((f) => ({ ...f, slug: e.target.value }))}
              className="h-8 text-xs font-mono"
            />
            <Input
              placeholder="描述（可选）"
              value={form.description}
              onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              className="h-8 text-xs"
            />
            <Input
              placeholder="图标 URL（可选）"
              value={form.icon}
              onChange={(e) => setForm((f) => ({ ...f, icon: e.target.value }))}
              className="h-8 text-xs"
            />
            <div className="flex items-center justify-between">
              <div>{error && <span className="text-xs text-destructive">{error}</span>}</div>
              <Button type="submit" size="sm" disabled={saving}>
                {saving ? "..." : "创建"}
              </Button>
            </div>
          </form>
        </Card>
      )}

      {apps.map((app) => (
        <Card
          key={app.id}
          className="flex items-center justify-between cursor-pointer hover:border-primary/50 transition-colors"
          onClick={() => navigate(`/dashboard/apps/${app.id}`)}
        >
          <div className="flex items-center gap-3">
            {app.icon ? (
              <img src={app.icon} alt="" className="w-8 h-8 rounded-lg object-cover" />
            ) : (
              <div className="w-8 h-8 rounded-lg bg-secondary flex items-center justify-center">
                <Blocks className="w-4 h-4 text-muted-foreground" />
              </div>
            )}
            <div>
              <p className="font-medium text-sm">{app.name}</p>
              <p className="text-xs text-muted-foreground font-mono mt-0.5">{app.slug}</p>
              {app.description && (
                <p className="text-xs text-muted-foreground mt-0.5 line-clamp-1">
                  {app.description}
                </p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            {app.commands?.length > 0 && (
              <span className="text-xs text-muted-foreground">{app.commands.length} 个命令</span>
            )}
            <Badge variant={app.status === "active" ? "default" : "outline"}>
              {app.status === "active" ? "启用" : app.status || "草稿"}
            </Badge>
          </div>
        </Card>
      ))}

      {apps.length === 0 && !creating && (
        <p className="text-center text-sm text-muted-foreground py-8">
          点击上方按钮创建你的第一个 App
        </p>
      )}
    </div>
  );
}
