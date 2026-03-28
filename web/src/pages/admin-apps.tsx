import { useEffect, useState } from "react";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent } from "../components/ui/dialog";
import { api } from "../lib/api";
import { Blocks, Trash2, X, Pencil } from "lucide-react";
import { useConfirm } from "@/components/ui/confirm-dialog";

export function AdminAppsTab() {
  const [apps, setApps] = useState<any[]>([]);
  const [selected, setSelected] = useState<any>(null);
  const [editing, setEditing] = useState(false);
  const [error, setError] = useState("");
  const { confirm, ConfirmDialog } = useConfirm();

  async function load() {
    try {
      setApps((await api.adminListApps()) || []);
    } catch {}
  }
  useEffect(() => {
    load();
  }, []);

  function openDetail(app: any) {
    setSelected(app);
    setEditing(false);
  }

  async function toggleListing(e: React.MouseEvent, app: any) {
    e.stopPropagation();
    const newListing = app.listing === "listed" ? "unlisted" : "listed";
    try {
      await api.setAppListing(app.id, newListing);
      load();
    } catch (err: any) {
      setError(err.message);
    }
  }

  async function handleDelete(app: any) {
    const ok = await confirm({
      title: "删除确认",
      description: `删除 App "${app.name}"？此操作不可恢复。`,
      confirmText: "删除",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await api.deleteApp(app.id);
      setSelected(null);
      load();
    } catch (err: any) {
      setError(err.message);
    }
  }

  async function handleApprove(app: any) {
    try {
      await api.reviewListing(app.id, true);
      load();
    } catch (err: any) {
      setError(err.message);
    }
  }

  async function handleReject(app: any) {
    const reason = prompt("拒绝原因：");
    if (!reason) return;
    try {
      await api.reviewListing(app.id, false, reason);
      load();
    } catch (err: any) {
      setError(err.message);
    }
  }

  return (
    <div className="space-y-3">
      {ConfirmDialog}
      {error ? <p className="text-xs text-destructive">{error}</p> : null}
      <div className="space-y-1">
        {apps.map((app) => (
          <div
            key={app.id}
            onClick={() => openDetail(app)}
            className={`flex items-center justify-between p-2.5 rounded-lg border cursor-pointer hover:border-primary/50 ${selected?.id === app.id ? "border-primary bg-primary/5" : "bg-card"}`}
          >
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-lg bg-secondary flex items-center justify-center text-base">
                {app.icon || <Blocks className="w-4 h-4 text-muted-foreground" />}
              </div>
              <div>
                <div className="flex items-center gap-1.5">
                  <span className="text-xs font-medium">{app.name}</span>
                  <span className="text-xs text-muted-foreground font-mono">{app.slug}</span>
                  {app.listing === "listed" ? (
                    <Badge variant="default" className="text-[10px]">
                      已上架
                    </Badge>
                  ) : null}
                  {app.listing === "pending" ? (
                    <Badge
                      variant="outline"
                      className="text-[10px] text-orange-500 border-orange-500"
                    >
                      待审核
                    </Badge>
                  ) : null}
                  {app.listing === "rejected" ? (
                    <Badge variant="destructive" className="text-[10px]">
                      已拒绝
                    </Badge>
                  ) : null}
                </div>
                <p className="text-xs text-muted-foreground">
                  {app.owner_name && `by ${app.owner_name} · `}
                  {(app.tools || []).length} 工具 · {(app.events || []).length} 事件
                </p>
              </div>
            </div>
            <div className="flex items-center gap-1 shrink-0">
              {app.listing === "pending" ? (
                <>
                  <Button
                    size="xs"
                    variant="ghost"
                    className="text-primary bg-primary/10 hover:bg-primary/20"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleApprove(app);
                    }}
                  >
                    通过
                  </Button>
                  <Button
                    size="xs"
                    variant="ghost"
                    className="text-destructive bg-destructive/10 hover:bg-destructive/20"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleReject(app);
                    }}
                  >
                    拒绝
                  </Button>
                </>
              ) : null}
              <Button
                size="xs"
                variant={app.listing === "listed" ? "ghost" : "secondary"}
                className={
                  app.listing === "listed" ? "text-primary bg-primary/10 hover:bg-primary/20" : ""
                }
                onClick={(e) => toggleListing(e, app)}
              >
                {app.listing === "listed" ? "下架" : "上架"}
              </Button>
            </div>
          </div>
        ))}
      </div>
      {apps.length === 0 ? (
        <p className="text-center text-sm text-muted-foreground py-8">暂无 App</p>
      ) : null}

      <Dialog
        open={!!selected}
        onOpenChange={(open) => {
          if (!open) setSelected(null);
        }}
      >
        <DialogContent className="max-w-lg max-h-[80vh] overflow-y-auto p-0">
          {selected ? (
            editing ? (
              <AppEditForm
                app={selected}
                onSave={() => {
                  setEditing(false);
                  load();
                  setSelected(null);
                }}
                onCancel={() => setEditing(false)}
              />
            ) : (
              <AppDetailView
                app={selected}
                onEdit={() => setEditing(true)}
                onDelete={() => handleDelete(selected)}
                onClose={() => setSelected(null)}
                onToggleListing={() => {
                  const newListing = selected.listing === "listed" ? "unlisted" : "listed";
                  api.setAppListing(selected.id, newListing).then(() => {
                    load();
                    setSelected({ ...selected, listing: newListing });
                  });
                }}
              />
            )
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function AppDetailView({
  app,
  onEdit,
  onDelete,
  onClose,
  onToggleListing,
}: {
  app: any;
  onEdit: () => void;
  onDelete: () => void;
  onClose: () => void;
  onToggleListing: () => void;
}) {
  const tools = (app.tools || []) as any[];
  const events = (app.events || []) as string[];
  const scopes = (app.scopes || []) as string[];

  return (
    <>
      <div className="p-4 border-b">
        <div className="flex items-center gap-2">
          {app.icon ? <span className="text-lg">{app.icon}</span> : null}
          <span className="font-semibold">{app.name}</span>
          <span className="text-xs text-muted-foreground font-mono">{app.slug}</span>
        </div>
        <p className="text-xs text-muted-foreground mt-0.5">{app.description || "无描述"}</p>
        <div className="flex gap-3 mt-1 text-xs text-muted-foreground">
          {app.owner_name ? <span>拥有者: {app.owner_name}</span> : null}
          {app.homepage ? (
            <a
              href={app.homepage}
              target="_blank"
              rel="noopener"
              className="text-primary hover:underline"
            >
              主页
            </a>
          ) : null}
          <span>{new Date(app.created_at * 1000).toLocaleDateString()}</span>
        </div>
      </div>

      {tools.length > 0 ? (
        <div className="p-4 border-b space-y-2">
          <p className="text-xs font-medium">工具 ({tools.length})</p>
          {tools.map((t: any, i: number) => (
            <div key={i} className="text-xs p-2 rounded border bg-card space-y-0.5">
              <div className="flex items-center gap-2">
                <code className="font-mono font-medium">{t.name}</code>
                {t.command ? (
                  <Badge variant="outline" className="text-[10px] font-mono">
                    /{t.command}
                  </Badge>
                ) : null}
              </div>
              <p className="text-muted-foreground">{t.description}</p>
              {t.parameters ? (
                <pre className="text-[10px] font-mono text-muted-foreground mt-1 overflow-x-auto">
                  {typeof t.parameters === "string"
                    ? t.parameters
                    : JSON.stringify(t.parameters, null, 2)}
                </pre>
              ) : null}
            </div>
          ))}
        </div>
      ) : null}

      {events.length > 0 || scopes.length > 0 ? (
        <div className="p-4 border-b space-y-2">
          {events.length > 0 ? (
            <div>
              <p className="text-xs font-medium mb-1">事件订阅</p>
              <div className="flex flex-wrap gap-1">
                {events.map((e) => (
                  <Badge key={e} variant="outline" className="text-[10px] font-mono">
                    {e}
                  </Badge>
                ))}
              </div>
            </div>
          ) : null}
          {scopes.length > 0 ? (
            <div>
              <p className="text-xs font-medium mb-1">权限</p>
              <div className="flex flex-wrap gap-1">
                {scopes.map((s) => (
                  <Badge key={s} variant="secondary" className="text-[10px] font-mono">
                    {s}
                  </Badge>
                ))}
              </div>
            </div>
          ) : null}
        </div>
      ) : null}

      <div className="p-4 flex justify-between">
        <div className="flex gap-2">
          <Button variant="destructive" size="sm" onClick={onDelete}>
            <Trash2 className="w-3.5 h-3.5 mr-1" /> 删除
          </Button>
          <Button variant="outline" size="sm" onClick={onToggleListing}>
            {app.listing === "listed" ? "下架" : "上架"}
          </Button>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={onEdit}>
            <Pencil className="w-3.5 h-3.5 mr-1" /> 编辑
          </Button>
          <Button variant="outline" size="sm" onClick={onClose}>
            关闭
          </Button>
        </div>
      </div>
    </>
  );
}

function AppEditForm({
  app,
  onSave,
  onCancel,
}: {
  app: any;
  onSave: () => void;
  onCancel: () => void;
}) {
  const [form, setForm] = useState({
    name: app.name || "",
    description: app.description || "",
    icon: app.icon || "",
    homepage: app.homepage || "",
    tools: JSON.stringify(app.tools || [], null, 2),
    events: JSON.stringify(app.events || [], null, 2),
    scopes: JSON.stringify(app.scopes || [], null, 2),
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function handleSave() {
    setSaving(true);
    setError("");
    try {
      const tools = JSON.parse(form.tools);
      const events = JSON.parse(form.events);
      const scopes = JSON.parse(form.scopes);
      await api.updateApp(app.id, {
        name: form.name,
        description: form.description,
        icon: form.icon,
        homepage: form.homepage,
        tools,
        events,
        scopes,
      });
      onSave();
    } catch (err: any) {
      setError(err.message || "JSON 格式错误");
    }
    setSaving(false);
  }

  return (
    <div className="p-4 space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium">编辑 App</h3>
        <Button variant="ghost" size="icon-xs" onClick={onCancel}>
          <X className="w-4 h-4" />
        </Button>
      </div>

      <div className="space-y-2">
        <Input
          placeholder="名称"
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          className="h-8 text-xs"
        />
        <Input
          placeholder="描述"
          value={form.description}
          onChange={(e) => setForm({ ...form, description: e.target.value })}
          className="h-8 text-xs"
        />
        <div className="flex gap-2">
          <Input
            placeholder="图标（emoji）"
            value={form.icon}
            onChange={(e) => setForm({ ...form, icon: e.target.value })}
            className="h-8 text-xs w-24"
          />
          <Input
            placeholder="主页 URL"
            value={form.homepage}
            onChange={(e) => setForm({ ...form, homepage: e.target.value })}
            className="h-8 text-xs flex-1"
          />
        </div>
      </div>

      <div className="space-y-1">
        <label className="text-xs font-medium">工具 (JSON)</label>
        <textarea
          value={form.tools}
          onChange={(e) => setForm({ ...form, tools: e.target.value })}
          rows={6}
          className="w-full rounded-md border border-input bg-transparent px-2 py-1 text-[11px] font-mono placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 resize-none"
        />
      </div>

      <div className="space-y-1">
        <label className="text-xs font-medium">事件 (JSON)</label>
        <textarea
          value={form.events}
          onChange={(e) => setForm({ ...form, events: e.target.value })}
          rows={2}
          className="w-full rounded-md border border-input bg-transparent px-2 py-1 text-[11px] font-mono placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 resize-none"
        />
      </div>

      <div className="space-y-1">
        <label className="text-xs font-medium">权限 (JSON)</label>
        <textarea
          value={form.scopes}
          onChange={(e) => setForm({ ...form, scopes: e.target.value })}
          rows={2}
          className="w-full rounded-md border border-input bg-transparent px-2 py-1 text-[11px] font-mono placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 resize-none"
        />
      </div>

      {error ? <p className="text-xs text-destructive">{error}</p> : null}

      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" onClick={onCancel}>
          取消
        </Button>
        <Button size="sm" onClick={handleSave} disabled={saving}>
          {saving ? "..." : "保存"}
        </Button>
      </div>
    </div>
  );
}
