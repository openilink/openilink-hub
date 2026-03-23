import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Card } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { api } from "../lib/api";
import { Github, Download, Check, X, Trash2, Send, ArrowLeft, ExternalLink, BookOpen, Bot, Puzzle, Shield } from "lucide-react";

const statusMap: Record<string, { label: string; variant: "default" | "outline" | "destructive" }> = {
  approved: { label: "已通过", variant: "default" },
  pending: { label: "待审核", variant: "outline" },
  rejected: { label: "已拒绝", variant: "destructive" },
};

export function PluginsPage({ embedded }: { embedded?: boolean }) {
  const [plugins, setPlugins] = useState<any[]>([]);
  const [tab, setTab] = useState<"marketplace" | "submit" | "review">("marketplace");
  const [user, setUser] = useState<any>(null);

  async function load() {
    try { setUser(await api.me()); } catch {}
    try {
      setPlugins(await api.listPlugins(tab === "review" ? "pending" : "approved") || []);
    } catch { setPlugins([]); }
  }

  useEffect(() => { load(); }, [tab]);

  const isLoggedIn = !!user;
  const isAdmin = user?.role === "admin";

  const content = (
    <div className="space-y-5">
        {/* Hero banner */}
        <div className="rounded-xl border bg-card p-5 space-y-3">
          <div className="flex items-start justify-between gap-4">
            <div>
              <h1 className="text-base font-semibold">社区驱动的 Webhook 插件</h1>
              <p className="text-xs text-muted-foreground mt-1">
                一键安装到你的渠道，自动转发消息到飞书、Slack、钉钉等服务。所有插件代码公开审核，在安全沙箱中执行。
              </p>
            </div>
            <div className="flex gap-2 shrink-0">
              {isLoggedIn && (
                <Button variant="outline" size="sm" className="text-xs" onClick={() => setTab("submit")}>
                  <Send className="w-3 h-3 mr-1" /> 提交插件
                </Button>
              )}
            </div>
          </div>

          {/* AI development callout */}
          <div className="flex items-start gap-3 p-3 rounded-lg bg-primary/5 border border-primary/20">
            <Bot className="w-5 h-5 text-primary shrink-0 mt-0.5" />
            <div>
              <p className="text-xs font-medium">使用 AI 编写插件</p>
              <p className="text-[11px] text-muted-foreground mt-0.5">
                将以下链接发送给你的 AI 助手（Claude、ChatGPT 等），它可以直接阅读并为你生成符合规范的插件代码：
              </p>
              <div className="flex items-center gap-2 mt-1.5">
                <code className="text-[10px] font-mono bg-background border rounded px-2 py-1 select-all">
                  {location.origin}/api/webhook-plugins/skill.md
                </code>
                <CopyButton value={`${location.origin}/api/webhook-plugins/skill.md`} />
                <a href="/api/webhook-plugins/skill.md" target="_blank" rel="noopener" className="text-[10px] text-primary hover:underline flex items-center gap-0.5">
                  <BookOpen className="w-3 h-3" /> 预览文档
                </a>
              </div>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border rounded-lg overflow-hidden w-fit">
          <button className={`px-3 py-1 text-xs cursor-pointer ${tab === "marketplace" ? "bg-secondary" : "text-muted-foreground"}`} onClick={() => setTab("marketplace")}>
            市场 {plugins.length > 0 && tab === "marketplace" ? `(${plugins.length})` : ""}
          </button>
          {isLoggedIn && (
            <button className={`px-3 py-1 text-xs cursor-pointer ${tab === "submit" ? "bg-secondary" : "text-muted-foreground"}`} onClick={() => setTab("submit")}>提交插件</button>
          )}
          {isAdmin && (
            <button className={`px-3 py-1 text-xs cursor-pointer ${tab === "review" ? "bg-secondary" : "text-muted-foreground"}`} onClick={() => setTab("review")}>
              审核 {plugins.length > 0 && tab === "review" ? `(${plugins.length})` : ""}
            </button>
          )}
        </div>

        {/* Content */}
        {tab === "marketplace" && (
          <div className="space-y-3">
            {plugins.length === 0 && (
              <div className="text-center py-16 space-y-3">
                <Puzzle className="w-10 h-10 mx-auto text-muted-foreground/50" />
                <p className="text-sm text-muted-foreground">暂无已审核的插件</p>
                {isLoggedIn && (
                  <Button variant="outline" size="sm" className="text-xs" onClick={() => setTab("submit")}>
                    成为第一个贡献者
                  </Button>
                )}
              </div>
            )}
            {plugins.map((p) => <PluginCard key={p.id} plugin={p} onRefresh={load} isAdmin={isAdmin} isLoggedIn={isLoggedIn} mode="marketplace" />)}
          </div>
        )}

        {tab === "submit" && <SubmitForm onSubmitted={() => { setTab("marketplace"); load(); }} />}

        {tab === "review" && (
          <div className="space-y-4">
            {plugins.length === 0 && (
              <div className="text-center py-12">
                <Shield className="w-8 h-8 mx-auto text-muted-foreground/50 mb-2" />
                <p className="text-sm text-muted-foreground">没有待审核的插件</p>
              </div>
            )}
            {plugins.map((p) => <ReviewCard key={p.id} plugin={p} onRefresh={load} />)}
          </div>
        )}
    </div>
  );

  if (embedded) return content;

  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b px-6 py-3 flex items-center justify-between shrink-0">
        <div className="flex items-center gap-2">
          <Link to="/" className="text-muted-foreground hover:text-foreground"><ArrowLeft className="w-4 h-4" /></Link>
          <Puzzle className="w-4 h-4 text-primary" />
          <span className="font-semibold text-sm">Webhook 插件市场</span>
        </div>
        <div className="flex items-center gap-2">
          {!isLoggedIn && <Link to="/login"><Button size="sm" className="text-xs">登录</Button></Link>}
          {isLoggedIn && <span className="text-xs text-muted-foreground">{user.username}</span>}
        </div>
      </header>
      <main className="flex-1 p-6 max-w-4xl mx-auto w-full">
        {content}
      </main>
      <footer className="border-t py-3 text-center text-[10px] text-muted-foreground">
        <a href="https://github.com/openilink/openilink-hub" target="_blank" rel="noopener" className="hover:text-primary">OpenILink Hub</a>
        {" · "}Webhook 插件运行在安全沙箱中（5s 超时 · 禁止系统访问 · 管理员审核）
      </footer>
    </div>
  );
}

function CopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button onClick={() => { navigator.clipboard.writeText(value); setCopied(true); setTimeout(() => setCopied(false), 2000); }}
      className="cursor-pointer text-muted-foreground hover:text-foreground">
      {copied ? <Check className="w-3 h-3 text-primary" /> : <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>}
    </button>
  );
}

function PluginCard({ plugin, onRefresh, isAdmin, isLoggedIn, mode }: {
  plugin: any; onRefresh: () => void; isAdmin: boolean; isLoggedIn: boolean; mode: string;
}) {
  const [detail, setDetail] = useState<any>(null);
  const [showScript, setShowScript] = useState(false);

  async function handleInstall() {
    const data = await api.installPlugin(plugin.id);
    await navigator.clipboard.writeText(data.script);
    alert("脚本已复制到剪贴板！\n\n推荐方式：进入渠道 → Webhook → 插件市场 → 选择此插件一键安装");
    onRefresh();
  }

  async function handleReview(status: string) {
    let reason = "";
    if (status === "rejected") {
      reason = prompt("请输入拒绝原因：") || "";
      if (!reason) return;
    }
    await api.reviewPlugin(plugin.id, status, reason);
    onRefresh();
  }

  async function handleDelete() {
    if (!confirm("确认删除此插件？")) return;
    await api.deletePlugin(plugin.id);
    onRefresh();
  }

  async function toggleScript() {
    if (!detail) { try { setDetail(await api.getPlugin(plugin.id)); } catch {} }
    setShowScript(!showScript);
  }

  const s = statusMap[plugin.status] || statusMap.pending;
  const config = plugin.config_schema || [];
  const grants = (plugin.grant_perms || "").split(",").filter(Boolean);
  const matchTypes = plugin.match_types || "*";
  const connectDomains = plugin.connect_domains || "*";
  const riskLevel = connectDomains === "*" && grants.includes("reply") ? "high"
    : connectDomains === "*" || grants.includes("reply") ? "medium" : "low";
  const riskColors: Record<string, string> = { low: "text-primary", medium: "text-yellow-500", high: "text-destructive" };
  const riskLabels: Record<string, string> = { low: "低风险", medium: "中风险", high: "高风险" };

  return (
    <Card className="space-y-2">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            {plugin.icon && <span className="text-base">{plugin.icon}</span>}
            <span className="font-medium text-sm">{plugin.name}</span>
            <Badge variant={s.variant} className="text-[10px]">{s.label}</Badge>
            <span className="text-[10px] text-muted-foreground">v{plugin.version}</span>
            {plugin.license && <span className="text-[10px] text-muted-foreground">{plugin.license}</span>}
          </div>
          <p className="text-xs text-muted-foreground mt-0.5">{plugin.description}</p>
          <div className="flex items-center gap-3 mt-1 text-[10px] text-muted-foreground flex-wrap">
            <span>作者: {plugin.author || "anonymous"}</span>
            {plugin.submitter_name && <span>拥有者: {plugin.submitter_name}</span>}
            {plugin.reviewer_name && <span>审核: {plugin.reviewer_name}</span>}
            <span>{plugin.install_count} 次安装</span>
            <span>{new Date(plugin.created_at * 1000).toLocaleDateString()}</span>
            {(plugin.github_url || plugin.homepage) && (
              <a href={plugin.homepage || plugin.github_url} target="_blank" rel="noopener" className="flex items-center gap-0.5 hover:text-primary">
                <Github className="w-3 h-3" /> 源码
              </a>
            )}
            {plugin.commit_hash && <span className="font-mono">{plugin.commit_hash.slice(0, 7)}</span>}
          </div>
          {/* Security summary */}
          <div className="flex items-center gap-2 mt-1 text-[10px] flex-wrap">
            <span className={riskColors[riskLevel]}><Shield className="w-3 h-3 inline mr-0.5" />{riskLabels[riskLevel]}</span>
            <span className="text-muted-foreground">权限: {grants.length > 0 ? grants.join(", ") : "none"}</span>
            <span className="text-muted-foreground">消息: {matchTypes}</span>
            {connectDomains !== "*" && <span className="text-muted-foreground">域名: {connectDomains}</span>}
          </div>
          {plugin.reject_reason && <p className="text-[10px] text-destructive mt-0.5">拒绝原因：{plugin.reject_reason}</p>}
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <Button size="sm" variant="ghost" className="h-7 text-xs" onClick={toggleScript}>
            {showScript ? "收起" : "源码"}
          </Button>
          {mode === "marketplace" && plugin.status === "approved" && isLoggedIn && (
            <Button size="sm" variant="outline" className="h-7 text-xs" onClick={handleInstall}>
              <Download className="w-3 h-3 mr-1" /> 安装
            </Button>
          )}
          {mode === "review" && (
            <>
              <Button size="sm" className="h-7 text-xs" onClick={() => handleReview("approved")}>
                <Check className="w-3 h-3 mr-1" /> 通过
              </Button>
              <Button size="sm" variant="destructive" className="h-7 text-xs" onClick={() => handleReview("rejected")}>
                <X className="w-3 h-3 mr-1" /> 拒绝
              </Button>
            </>
          )}
          {isAdmin && (
            <Button size="sm" variant="ghost" className="h-7" onClick={handleDelete}>
              <Trash2 className="w-3 h-3 text-destructive" />
            </Button>
          )}
        </div>
      </div>

      {config.length > 0 && (
        <div className="text-[10px] text-muted-foreground">
          配置项：{config.map((c: any) => `${c.name} (${c.description || c.type})`).join("、")}
        </div>
      )}

      {showScript && (
        <div>
          {detail?.script ? (
            <pre className="text-[10px] bg-background border rounded p-3 overflow-x-auto max-h-64 overflow-y-auto whitespace-pre-wrap font-mono">{detail.script}</pre>
          ) : plugin.github_url ? (
            <a href={plugin.github_url} target="_blank" rel="noopener" className="text-xs text-primary flex items-center gap-1">
              <ExternalLink className="w-3 h-3" /> 在 GitHub 查看源码
            </a>
          ) : (
            <p className="text-[10px] text-muted-foreground">登录后可查看脚本源码</p>
          )}
        </div>
      )}
    </Card>
  );
}

function ReviewCard({ plugin, onRefresh }: { plugin: any; onRefresh: () => void }) {
  const [detail, setDetail] = useState<any>(null);
  const [rejectReason, setRejectReason] = useState("");
  const [showReject, setShowReject] = useState(false);

  useEffect(() => {
    api.getPlugin(plugin.id).then(setDetail).catch(() => {});
  }, [plugin.id]);

  async function handleApprove() {
    await api.reviewPlugin(plugin.id, "approved");
    onRefresh();
  }
  async function handleReject() {
    if (!rejectReason.trim()) return;
    await api.reviewPlugin(plugin.id, "rejected", rejectReason.trim());
    onRefresh();
  }
  async function handleDelete() {
    if (!confirm("永久删除此插件？")) return;
    await api.deletePlugin(plugin.id);
    onRefresh();
  }

  const grants = (plugin.grant_perms || "").split(",").filter(Boolean);
  const matchTypes = plugin.match_types || "*";
  const connectDomains = plugin.connect_domains || "*";
  const hasReply = grants.includes("reply");
  const hasSkip = grants.includes("skip");
  const isGrantNone = grants.includes("none");
  const wildcardConnect = connectDomains === "*";
  const wildcardMatch = matchTypes === "*";

  const risks: { level: "ok" | "warn" | "danger"; text: string }[] = [];
  if (isGrantNone) risks.push({ level: "ok", text: "声明 @grant none — 无副作用" });
  else if (grants.length === 0) risks.push({ level: "warn", text: "未声明 @grant — 默认全部 API 可用" });
  if (hasReply) risks.push({ level: "warn", text: "使用 reply() — 可向用户发送消息" });
  if (hasSkip) risks.push({ level: "ok", text: "使用 skip() — 可跳过 webhook 推送" });
  if (wildcardConnect) risks.push({ level: "danger", text: "@connect * — 可将请求重定向到任意域名" });
  else if (connectDomains) risks.push({ level: "ok", text: `@connect 限定域名: ${connectDomains}` });
  if (wildcardMatch) risks.push({ level: "ok", text: "@match * — 所有消息类型触发" });
  else risks.push({ level: "ok", text: `@match 限定类型: ${matchTypes}` });

  // Check script for suspicious patterns
  const scriptText = detail?.script || "";
  if (scriptText.includes("while(true)") || scriptText.includes("for(;;)")) risks.push({ level: "danger", text: "检测到疑似死循环" });
  if (scriptText.includes("__proto__") || scriptText.includes("prototype")) risks.push({ level: "warn", text: "检测到原型链操作" });
  if ((scriptText.match(/reply\(/g) || []).length > 3) risks.push({ level: "warn", text: `多处 reply() 调用 (${(scriptText.match(/reply\(/g) || []).length} 处)` });

  const riskColors = { ok: "text-primary", warn: "text-yellow-500", danger: "text-destructive" };
  const riskIcons = { ok: "✓", warn: "⚠", danger: "✕" };
  const overallRisk = risks.some(r => r.level === "danger") ? "danger" : risks.some(r => r.level === "warn") ? "warn" : "ok";
  const overallLabels = { ok: "低风险", warn: "需注意", danger: "高风险" };
  const overallColors = { ok: "border-primary/30 bg-primary/5", warn: "border-yellow-500/30 bg-yellow-500/5", danger: "border-destructive/30 bg-destructive/5" };

  return (
    <div className={`rounded-xl border-2 ${overallColors[overallRisk]} p-4 space-y-3`}>
      {/* Header */}
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2 flex-wrap">
            {plugin.icon && <span className="text-lg">{plugin.icon}</span>}
            <span className="font-semibold text-sm">{plugin.name}</span>
            <span className="text-[10px] text-muted-foreground">v{plugin.version}</span>
            {plugin.namespace && <span className="text-[10px] font-mono text-muted-foreground">{plugin.namespace}</span>}
          </div>
          <p className="text-xs text-muted-foreground mt-0.5">{plugin.description}</p>
          <div className="flex items-center gap-3 mt-1 text-[10px] text-muted-foreground flex-wrap">
            <span>作者: {plugin.author || "anonymous"}</span>
            <span>拥有者: {plugin.submitter_name}</span>
            {plugin.license && <span>{plugin.license}</span>}
            {plugin.github_url && (
              <a href={plugin.github_url} target="_blank" rel="noopener" className="text-primary hover:underline flex items-center gap-0.5">
                <Github className="w-3 h-3" /> GitHub
              </a>
            )}
            {plugin.commit_hash && <span className="font-mono">{plugin.commit_hash.slice(0, 7)}</span>}
          </div>
        </div>
        <div className={`px-2 py-1 rounded text-xs font-medium ${riskColors[overallRisk]}`}>
          <Shield className="w-3.5 h-3.5 inline mr-0.5" />
          {overallLabels[overallRisk]}
        </div>
      </div>

      {/* Security analysis */}
      <div className="rounded-lg border bg-card p-3 space-y-1.5">
        <p className="text-xs font-medium flex items-center gap-1"><Shield className="w-3.5 h-3.5" /> 安全分析</p>
        {risks.map((r, i) => (
          <div key={i} className={`text-[11px] flex items-start gap-1.5 ${riskColors[r.level]}`}>
            <span className="shrink-0">{riskIcons[r.level]}</span>
            <span>{r.text}</span>
          </div>
        ))}
      </div>

      {/* Config schema */}
      {(plugin.config_schema || []).length > 0 && (
        <div className="rounded-lg border bg-card p-3">
          <p className="text-xs font-medium mb-1">配置参数</p>
          <div className="space-y-1">
            {(plugin.config_schema || []).map((c: any, i: number) => (
              <div key={i} className="text-[11px] flex items-center gap-2">
                <code className="font-mono bg-background px-1 rounded">{c.name}</code>
                <span className="text-muted-foreground">{c.type}</span>
                {c.description && <span className="text-muted-foreground">— {c.description}</span>}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Source code */}
      <div className="rounded-lg border bg-card">
        <div className="px-3 py-2 border-b flex items-center justify-between">
          <p className="text-xs font-medium">源码</p>
          <span className="text-[10px] text-muted-foreground">{scriptText.split("\n").length} 行</span>
        </div>
        <pre className="p-3 text-[10px] font-mono overflow-x-auto max-h-80 overflow-y-auto whitespace-pre-wrap">
          {scriptText || "加载中..."}
        </pre>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2 pt-1">
        {!showReject ? (
          <>
            <Button size="sm" onClick={handleApprove} className="flex-1">
              <Check className="w-3.5 h-3.5 mr-1" /> 通过审核
            </Button>
            <Button size="sm" variant="outline" onClick={() => setShowReject(true)} className="flex-1">
              <X className="w-3.5 h-3.5 mr-1" /> 拒绝
            </Button>
            <Button size="sm" variant="ghost" onClick={handleDelete}>
              <Trash2 className="w-3.5 h-3.5 text-destructive" />
            </Button>
          </>
        ) : (
          <div className="flex-1 space-y-2">
            <Input value={rejectReason} onChange={(e) => setRejectReason(e.target.value)}
              placeholder="请输入拒绝原因..." className="h-8 text-xs" autoFocus />
            <div className="flex gap-2">
              <Button size="sm" variant="destructive" onClick={handleReject} disabled={!rejectReason.trim()} className="flex-1">
                确认拒绝
              </Button>
              <Button size="sm" variant="ghost" onClick={() => { setShowReject(false); setRejectReason(""); }}>
                取消
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function SubmitForm({ onSubmitted }: { onSubmitted: () => void }) {
  const [mode, setMode] = useState<"github" | "paste">("github");
  const [url, setUrl] = useState("");
  const [script, setScript] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const data = mode === "github" ? { github_url: url.trim() } : { script: script.trim() };
    if (!data.github_url && !data.script) return;
    setSubmitting(true); setError("");
    try {
      await api.submitPlugin(data);
      setUrl(""); setScript("");
      onSubmitted();
    } catch (err: any) { setError(err.message); }
    setSubmitting(false);
  }

  const canSubmit = mode === "github" ? !!url.trim() : !!script.trim();

  const templateScript = `// ==WebhookPlugin==
// @name         我的插件
// @namespace    github.com/yourname
// @version      1.0.0
// @description  插件功能描述
// @author       你的名字
// @license      MIT
// @icon         🔔
// @match        text
// @connect      *
// @grant        none
// ==/WebhookPlugin==

function onRequest(ctx) {
  ctx.req.body = JSON.stringify({
    text: ctx.msg.sender + ": " + ctx.msg.content
  });
}`;

  return (
    <div className="space-y-4">
      {/* AI development tip */}
      <Card className="flex items-start gap-3 bg-primary/5 border-primary/20">
        <Bot className="w-5 h-5 text-primary shrink-0 mt-0.5" />
        <div className="text-xs">
          <p className="font-medium">推荐：让 AI 帮你写插件</p>
          <p className="text-muted-foreground mt-0.5">
            将 <a href="/api/webhook-plugins/skill.md" target="_blank" className="text-primary hover:underline">skill.md</a> 链接发给 AI 助手，描述你的需求，AI 会生成完整的插件代码。生成后粘贴到下方提交即可。
          </p>
        </div>
      </Card>

      <Card className="space-y-3">
        <h3 className="text-sm font-medium">提交 Webhook 插件</h3>

        <div className="flex border rounded-lg overflow-hidden w-fit">
          <button className={`px-3 py-1 text-xs cursor-pointer ${mode === "github" ? "bg-secondary" : "text-muted-foreground"}`} onClick={() => setMode("github")}>GitHub 链接</button>
          <button className={`px-3 py-1 text-xs cursor-pointer ${mode === "paste" ? "bg-secondary" : "text-muted-foreground"}`} onClick={() => setMode("paste")}>粘贴脚本</button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-2">
          {mode === "github" ? (
            <>
              <Input value={url} onChange={(e) => setUrl(e.target.value)}
                placeholder="https://github.com/user/repo/blob/main/plugin.js"
                className="h-8 text-xs font-mono" />
              <p className="text-[10px] text-muted-foreground">自动拉取脚本并固定 commit hash，确保审核的代码就是运行的代码。</p>
            </>
          ) : (
            <>
              <textarea value={script} onChange={(e) => setScript(e.target.value)}
                placeholder={templateScript}
                rows={16}
                className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-[11px] font-mono placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:border-ring resize-none" />
              <p className="text-[10px] text-muted-foreground">
                使用 <code className="bg-secondary px-1 rounded">{"// ==WebhookPlugin=="}</code> 格式声明元数据。
                <a href="/api/webhook-plugins/skill.md" target="_blank" className="text-primary hover:underline ml-1">查看完整规范</a>
              </p>
            </>
          )}
          <div className="flex items-center justify-between">
            {error && <span className="text-xs text-destructive">{error}</span>}
            <Button type="submit" size="sm" disabled={submitting || !canSubmit} className="ml-auto">
              <Send className="w-3.5 h-3.5 mr-1" /> {submitting ? "提交中..." : "提交审核"}
            </Button>
          </div>
        </form>
      </Card>

      {/* Quick reference */}
      <Card className="space-y-2">
        <h3 className="text-xs font-medium">快速参考</h3>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-2 text-[10px]">
          <div className="p-2 rounded border bg-background">
            <p className="font-medium mb-1">ctx.msg（消息）</p>
            <p className="text-muted-foreground">.sender .content .msg_type .channel_id .bot_id .timestamp .items[]</p>
          </div>
          <div className="p-2 rounded border bg-background">
            <p className="font-medium mb-1">ctx.req（请求）</p>
            <p className="text-muted-foreground">.url .method .headers .body</p>
          </div>
          <div className="p-2 rounded border bg-background">
            <p className="font-medium mb-1">全局函数</p>
            <p className="text-muted-foreground">reply(text) skip() JSON.parse/stringify</p>
          </div>
        </div>
        <div className="flex items-center gap-3 text-[10px] text-muted-foreground">
          <span><Shield className="w-3 h-3 inline" /> 5s 超时</span>
          <span>栈深 64</span>
          <span>禁止 eval/require</span>
          <span>reply 最多 10 次</span>
          <a href="/api/webhook-plugins/skill.md" target="_blank" className="text-primary hover:underline ml-auto flex items-center gap-0.5">
            <BookOpen className="w-3 h-3" /> 完整文档
          </a>
        </div>
      </Card>
    </div>
  );
}
