import { useEffect, useState } from "react";
import { useParams, Link, useNavigate, useLocation } from "react-router-dom";
import {
  ArrowLeft,
  Copy,
  Check,
  Cable,
  Bot as BotIcon,
  Webhook,
  Radio,
  RotateCw,
  Trash2,
  Settings,
  ShieldCheck,
  Zap,
  Activity,
  Code,
  Terminal,
  ExternalLink,
  Loader2,
  Info,
  ChevronRight,
  Filter,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { api, botDisplayName } from "../lib/api";
import { useBot, useBotChannels, useWebhookLogs } from "@/hooks/use-bots";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useToast } from "@/hooks/use-toast";
import { Skeleton } from "@/components/ui/skeleton";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { useConfirm } from "@/components/ui/confirm-dialog";

export function ChannelDetailPage() {
  const { id: botId, cid: channelId } = useParams<{ id: string; cid: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { toast } = useToast();
  const { confirm, ConfirmDialog } = useConfirm();

  const activeTab = location.pathname.split("/").pop() || "overview";

  const { data: bot, isLoading: botLoading, isError: botError } = useBot(botId || "");
  const { data: channels, isLoading: channelsLoading, isError: channelsError, refetch: refetchChannels } = useBotChannels(botId || "");
  const channel = (channels || []).find((c: any) => c.id === channelId) || null;
  const loading = botLoading || channelsLoading;
  const fetchError = botError || channelsError;

  function load() {
    refetchChannels();
  }

  async function handleDelete() {
    const ok = await confirm({
      title: "删除确认",
      description: "确定要删除此转发规则？API Key 将失效。",
      confirmText: "删除",
      variant: "destructive",
    });
    if (!ok) return;
    try {
      await api.deleteChannel(botId!, channelId!);
      toast({ title: "已删除" });
      navigate(`/dashboard/accounts/${botId}/channels`);
    } catch (e: any) {
      toast({ variant: "destructive", title: "删除失败", description: e.message });
    }
  }

  async function handleToggle() {
    try {
      const nextStatus = !channel.enabled;
      await api.updateChannel(botId!, channelId!, { enabled: nextStatus });
      toast({ title: nextStatus ? "已启用" : "已停用" });
      load();
    } catch (e: any) {
      toast({ variant: "destructive", title: "操作失败", description: e.message });
    }
  }

  if (loading)
    return (
      <div className="space-y-6">
        <Skeleton className="h-12 w-[300px]" />
        <Skeleton className="h-64 w-full" />
      </div>
    );

  if (!channel || !bot)
    return (
      <div className="flex flex-col items-center justify-center py-20 text-center">
        <Info className="h-10 w-10 text-muted-foreground opacity-20 mb-4" />
        <p className="text-sm font-medium text-muted-foreground">{fetchError ? "加载失败" : "未找到"}</p>
        <Button variant="link" asChild>
          <Link to={`/dashboard/accounts/${botId}/channels`}>返回列表</Link>
        </Button>
      </div>
    );

  return (
    <div className="flex flex-col gap-6">
      {ConfirmDialog}
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div className="flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 text-primary shrink-0 shadow-sm border border-primary/20">
            <Cable className="h-6 w-6" />
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-bold tracking-tight">{channel.name}</h1>
              <Button
                variant={channel.enabled ? "default" : "outline"}
                size="sm"
                className="h-6 px-2 text-[10px] uppercase font-bold rounded-full"
                onClick={handleToggle}
              >
                {channel.enabled ? "已启用" : "已停用"}
              </Button>
            </div>
            <div className="flex items-center gap-2 text-xs text-muted-foreground font-mono">
              <Link
                to={`/dashboard/accounts/${botId}`}
                className="hover:text-primary transition-colors flex items-center gap-1"
              >
                <BotIcon className="h-3 w-3" /> {botDisplayName(bot)}
              </Link>
              <ChevronRight className="h-3 w-3 opacity-30" />
              <span>规则 ID: {channel.id.slice(0, 8)}</span>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={load} className="h-9 w-9 p-0">
            <RotateCw className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={handleDelete}
            className="h-9 gap-2 text-destructive border-destructive/20 hover:bg-destructive/10"
          >
            <Trash2 className="h-4 w-4" /> <span className="hidden sm:inline">删除</span>
          </Button>
        </div>
      </div>

      <Tabs
        value={activeTab}
        onValueChange={(v: string) =>
          navigate(`/dashboard/accounts/${botId}/channel/${channelId}/${v}`)
        }
        className="space-y-6"
      >
        <TabsList className="bg-muted/50 p-1">
          <TabsTrigger value="overview" className="gap-2 px-4">
            <Activity className="h-4 w-4" /> 概览
          </TabsTrigger>
          <TabsTrigger value="webhook" className="gap-2 px-4">
            <Webhook className="h-4 w-4" /> Webhook
          </TabsTrigger>
          <TabsTrigger value="ai" className="gap-2 px-4">
            <BotIcon className="h-4 w-4" /> AI 回复
          </TabsTrigger>
          <TabsTrigger value="filter" className="gap-2 px-4">
            <Filter className="h-4 w-4" /> 过滤条件
          </TabsTrigger>
          <TabsTrigger value="logs" className="gap-2 px-4">
            <Terminal className="h-4 w-4" /> 请求日志
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="m-0 space-y-6">
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            <Card className="border-border/50 bg-card/50">
              <CardHeader className="pb-3">
                <CardTitle className="text-xs font-bold uppercase tracking-widest text-muted-foreground">
                  基本信息
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground">名称</p>
                  <p className="text-sm font-semibold">{channel.name}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground">@提及</p>
                  <div className="flex items-center gap-2">
                    <code className="text-xs bg-muted px-1.5 py-0.5 rounded font-mono">
                      @{channel.handle || "null"}
                    </code>
                    <p className="text-[10px] text-muted-foreground italic">
                      {channel.handle ? "仅匹配此提及" : "接收所有消息"}
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-border/50 bg-card/50">
              <CardHeader className="pb-3">
                <CardTitle className="text-xs font-bold uppercase tracking-widest text-muted-foreground">
                  API Key
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground">频道密钥</p>
                  <div className="flex items-center gap-2">
                    <code className="text-xs bg-muted px-1.5 py-0.5 rounded font-mono truncate max-w-[140px]">
                      {channel.api_key}
                    </code>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6"
                      onClick={() => {
                        navigator.clipboard.writeText(channel.api_key);
                        toast({ title: "已复制 API Key" });
                      }}
                    >
                      <Copy className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
                <div className="space-y-1">
                  <p className="text-xs text-muted-foreground">接口地址</p>
                  <p className="text-[10px] font-mono opacity-70 truncate">
                    {window.location.origin}/api/v1/channels
                  </p>
                </div>
              </CardContent>
            </Card>

            <Card className="border-border/50 bg-card/50 border-primary/20">
              <CardHeader className="pb-3">
                <CardTitle className="text-xs font-bold uppercase tracking-widest text-muted-foreground">
                  状态
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex flex-col gap-2">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground">最后活跃</span>
                    <span className="font-medium">
                      {channel.updated_at
                        ? new Date(channel.updated_at * 1000).toLocaleString()
                        : "从未"}
                    </span>
                  </div>
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground">Webhook 状态</span>
                    <Badge
                      variant={channel.webhook_config?.url ? "default" : "secondary"}
                      className="h-4 text-[9px]"
                    >
                      {channel.webhook_config?.url ? "已配置" : "未开启"}
                    </Badge>
                  </div>
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground">AI 增强</span>
                    <Badge
                      variant={channel.ai_config?.enabled ? "default" : "secondary"}
                      className="h-4 text-[9px]"
                    >
                      {channel.ai_config?.enabled ? "活跃" : "关闭"}
                    </Badge>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          <Card className="border-border/50">
            <CardHeader>
              <CardTitle className="text-sm">接入方式</CardTitle>
              <CardDescription>通过 WebSocket 或 HTTP 接收消息。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="rounded-lg bg-muted/50 p-4 border font-mono text-[11px] space-y-4 leading-relaxed">
                <div>
                  <p className="text-primary font-bold mb-1">// WebSocket（实时推送）</p>
                  <code className="block bg-background p-2 rounded border">{`ws://${window.location.host}/api/v1/channels/connect?key=${channel.api_key.slice(0, 12)}...`}</code>
                </div>
                <div>
                  <p className="text-primary font-bold mb-1">// HTTP POST（发送回复）</p>
                  <code className="block bg-background p-2 rounded border">{`POST ${window.location.origin}/api/v1/channels/send?key=${channel.api_key.slice(0, 12)}...`}</code>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="webhook" className="m-0">
          <WebhookTab channel={channel} botId={botId!} onRefresh={load} />
        </TabsContent>

        <TabsContent value="ai" className="m-0">
          <AITab channel={channel} botId={botId!} onRefresh={load} />
        </TabsContent>

        <TabsContent value="filter" className="m-0">
          <FilterTab channel={channel} botId={botId!} onRefresh={load} />
        </TabsContent>

        <TabsContent value="logs" className="m-0">
          <WebhookLogsTab channel={channel} botId={botId!} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// ==================== Sub-Tabs Components (Standardized) ====================

function WebhookTab({
  channel,
  botId,
  onRefresh,
}: {
  channel: any;
  botId: string;
  onRefresh: () => void;
}) {
  const cfg = channel.webhook_config || {};
  const [form, setForm] = useState({
    url: cfg.url || "",
    authType: cfg.auth?.type || "none",
    authToken: cfg.auth?.token || "",
    authName: cfg.auth?.name || "",
    authValue: cfg.auth?.value || cfg.auth?.secret || "",
    script: cfg.script || "",
  });
  const [saving, setSaving] = useState(false);
  const { toast } = useToast();

  async function handleSave() {
    setSaving(true);
    try {
      let auth: any = null;
      if (form.authType === "bearer") auth = { type: "bearer", token: form.authToken };
      else if (form.authType === "header")
        auth = { type: "header", name: form.authName, value: form.authValue };
      else if (form.authType === "hmac") auth = { type: "hmac", secret: form.authValue };

      await api.updateChannel(botId, channel.id, {
        webhook_config: {
          url: form.url,
          auth,
          script: form.script || undefined,
        },
      });
      toast({ title: "Webhook 配置已更新" });
      onRefresh();
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }

  return (
    <div className="grid gap-6 md:grid-cols-2">
      <Card className="border-border/50">
        <CardHeader>
          <CardTitle className="text-lg">Webhook 设置</CardTitle>
          <CardDescription>消息将转发到此地址。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <label className="text-xs font-medium">Webhook URL</label>
            <Input
              placeholder="https://..."
              value={form.url}
              onChange={(e) => setForm({ ...form, url: e.target.value })}
              className="font-mono"
            />
          </div>
          <div className="space-y-3 pt-2">
            <p className="text-xs font-medium">认证方式</p>
            <div className="flex flex-wrap gap-2">
              {["none", "bearer", "header", "hmac"].map((t) => (
                <Button
                  key={t}
                  variant={form.authType === t ? "default" : "outline"}
                  className="h-7 px-3 py-1 uppercase text-[10px] rounded-full"
                  onClick={() => setForm({ ...form, authType: t as any })}
                >
                  {t}
                </Button>
              ))}
            </div>
            <div className="pt-2">
              {form.authType === "bearer" ? (
                <Input
                  placeholder="Token"
                  value={form.authToken}
                  onChange={(e) => setForm({ ...form, authToken: e.target.value })}
                  className="h-9 font-mono"
                />
              ) : null}
              {form.authType === "header" ? (
                <div className="flex gap-2">
                  <Input
                    placeholder="Name"
                    value={form.authName}
                    onChange={(e) => setForm({ ...form, authName: e.target.value })}
                    className="h-9"
                  />
                  <Input
                    placeholder="Value"
                    value={form.authValue}
                    onChange={(e) => setForm({ ...form, authValue: e.target.value })}
                    className="h-9"
                  />
                </div>
              ) : null}
              {form.authType === "hmac" ? (
                <Input
                  placeholder="Secret"
                  value={form.authValue}
                  onChange={(e) => setForm({ ...form, authValue: e.target.value })}
                  className="h-9 font-mono"
                />
              ) : null}
            </div>
          </div>
        </CardContent>
        <CardFooter className="bg-muted/30 pt-4 flex justify-end">
          <Button onClick={handleSave} disabled={saving} size="sm">
            {saving ? <Loader2 className="mr-2 h-3 w-3 animate-spin" /> : null}保存设置
          </Button>
        </CardFooter>
      </Card>

      <Card className="border-border/50">
        <CardHeader>
          <CardTitle className="text-lg">脚本处理</CardTitle>
          <CardDescription>在转发前对消息做处理。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <textarea
              placeholder={`function onRequest(ctx) {\n  // 转换消息格式...\n}`}
              value={form.script}
              onChange={(e) => setForm({ ...form, script: e.target.value })}
              className="w-full h-40 bg-muted/30 border rounded-md p-3 font-mono text-[11px] focus:outline-none"
            />
            <p className="text-[10px] text-muted-foreground">
              脚本在安全沙箱中运行，支持 reply() 和修改 ctx.body。
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function AITab({
  channel,
  botId,
  onRefresh,
}: {
  channel: any;
  botId: string;
  onRefresh: () => void;
}) {
  const cfg = channel.ai_config || {};
  const [form, setForm] = useState({
    enabled: cfg.enabled || false,
    source: cfg.source || "builtin",
    baseUrl: cfg.base_url || "",
    apiKey: "",
    model: cfg.model || "",
    prompt: cfg.system_prompt || "",
    history: cfg.max_history || 20,
  });
  const [saving, setSaving] = useState(false);
  const { toast } = useToast();

  async function handleSave() {
    setSaving(true);
    try {
      await api.updateChannel(botId, channel.id, {
        ai_config: {
          ...form,
          base_url: form.baseUrl,
          api_key: form.apiKey || undefined,
          system_prompt: form.prompt,
          max_history: form.history,
        },
      });
      toast({ title: "AI 配置已保存" });
      onRefresh();
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }

  return (
    <Card className="border-border/50 max-w-3xl">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <CardTitle>AI 自动回复</CardTitle>
            <CardDescription>收到消息后自动用 AI 回复。</CardDescription>
          </div>
          <Button
            variant={form.enabled ? "default" : "outline"}
            size="sm"
            onClick={() => setForm({ ...form, enabled: !form.enabled })}
          >
            {form.enabled ? "已开启" : "已关闭"}
          </Button>
        </div>
      </CardHeader>
      {form.enabled ? (
        <CardContent className="space-y-6 pt-2 animate-in fade-in slide-in-from-top-2">
          <div className="space-y-3">
            <p className="text-xs font-medium">模型来源</p>
            <div className="flex gap-2">
              <Button
                variant={form.source === "builtin" ? "default" : "outline"}
                size="sm"
                onClick={() => setForm({ ...form, source: "builtin" })}
              >
                系统内置
              </Button>
              <Button
                variant={form.source === "custom" ? "default" : "outline"}
                size="sm"
                onClick={() => setForm({ ...form, source: "custom" })}
              >
                自定义接口 (OpenAI 协议)
              </Button>
            </div>
          </div>
          {form.source === "custom" ? (
            <div className="grid gap-4 sm:grid-cols-2 border rounded-xl p-4 bg-muted/20">
              <div className="sm:col-span-2 space-y-1.5">
                <label className="text-[11px] font-bold uppercase">接口地址</label>
                <Input
                  placeholder="https://api.openai.com/v1"
                  value={form.baseUrl}
                  onChange={(e) => setForm({ ...form, baseUrl: e.target.value })}
                  className="h-9 font-mono text-xs"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-[11px] font-bold uppercase">API Key</label>
                <Input
                  type="password"
                  placeholder="sk-..."
                  value={form.apiKey}
                  onChange={(e) => setForm({ ...form, apiKey: e.target.value })}
                  className="h-9 font-mono text-xs"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-[11px] font-bold uppercase">模型</label>
                <Input
                  placeholder="gpt-4o"
                  value={form.model}
                  onChange={(e) => setForm({ ...form, model: e.target.value })}
                  className="h-9 font-mono text-xs"
                />
              </div>
            </div>
          ) : null}
          <div className="space-y-2">
            <label className="text-xs font-medium">系统提示词</label>
            <textarea
              value={form.prompt}
              onChange={(e) => setForm({ ...form, prompt: e.target.value })}
              className="w-full h-24 bg-muted/30 border rounded-md p-3 text-xs leading-relaxed focus:outline-none"
              placeholder="你是一个智能助理..."
            />
          </div>
          <div className="flex items-center gap-4">
            <div className="space-y-1">
              <label className="text-xs font-medium">上下文深度</label>
              <Input
                type="number"
                value={form.history}
                onChange={(e) => setForm({ ...form, history: parseInt(e.target.value) || 20 })}
                className="w-24 h-9"
              />
            </div>
            <p className="text-[10px] text-muted-foreground italic flex-1">
              较大的深度会消耗更多 Token，但对话连贯性更好。
            </p>
          </div>
        </CardContent>
      ) : null}
      <CardFooter className="bg-muted/30 pt-4 flex justify-end">
        <Button onClick={handleSave} disabled={saving} size="sm">
          保存 AI 配置
        </Button>
      </CardFooter>
    </Card>
  );
}

function FilterTab({
  channel,
  botId,
  onRefresh,
}: {
  channel: any;
  botId: string;
  onRefresh: () => void;
}) {
  const rule = channel.filter_rule || {};
  const [vals, setVals] = useState({
    uids: (rule.user_ids || []).join(", "),
    words: (rule.keywords || []).join(", "),
    types: (rule.message_types || []).join(", "),
  });
  const [saving, setSaving] = useState(false);
  const { toast } = useToast();

  async function handleSave() {
    setSaving(true);
    const parse = (s: string) =>
      s
        .split(",")
        .map((v) => v.trim())
        .filter(Boolean);
    try {
      await api.updateChannel(botId, channel.id, {
        filter_rule: {
          user_ids: parse(vals.uids),
          keywords: parse(vals.words),
          message_types: parse(vals.types),
        },
      });
      toast({ title: "过滤规则已更新" });
      onRefresh();
    } catch (e: any) {
      toast({ variant: "destructive", title: "保存失败", description: e.message });
    }
    setSaving(false);
  }

  return (
    <Card className="border-border/50 max-w-2xl">
      <CardHeader>
        <CardTitle>过滤条件</CardTitle>
        <CardDescription>设置哪些消息会被转发。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-1.5">
          <label className="text-xs font-medium">用户白名单 (User ID)</label>
          <Input
            value={vals.uids}
            onChange={(e) => setVals({ ...vals, uids: e.target.value })}
            placeholder="u123, u456..."
            className="h-9 font-mono text-xs"
          />
        </div>
        <div className="space-y-1.5">
          <label className="text-xs font-medium">关键字包含</label>
          <Input
            value={vals.words}
            onChange={(e) => setVals({ ...vals, words: e.target.value })}
            placeholder="help, 菜单..."
            className="h-9 text-xs"
          />
        </div>
        <div className="space-y-1.5">
          <label className="text-xs font-medium">消息类型限制</label>
          <Input
            value={vals.types}
            onChange={(e) => setVals({ ...vals, types: e.target.value })}
            placeholder="text, image, voice..."
            className="h-9 font-mono text-xs"
          />
        </div>
        <div className="mt-4 p-3 bg-primary/5 rounded-lg border border-primary/10 flex gap-3 items-start">
          <Info className="h-4 w-4 text-primary shrink-0 mt-0.5" />
          <p className="text-[10px] text-muted-foreground leading-relaxed">
            用英文逗号分隔多个值。留空表示匹配所有消息。满足任一条件即转发。
          </p>
        </div>
      </CardContent>
      <CardFooter className="bg-muted/30 pt-4 flex justify-end">
        <Button onClick={handleSave} disabled={saving} size="sm">
          保存过滤规则
        </Button>
      </CardFooter>
    </Card>
  );
}

function WebhookLogsTab({ channel, botId }: { channel: any; botId: string }) {
  const { data: logs = [], isLoading: loading, isError: logsError, refetch } = useWebhookLogs(botId, channel.id);

  useEffect(() => {
    const t = setInterval(() => refetch(), 5000);
    return () => clearInterval(t);
  }, [botId, channel.id, refetch]);

  function load() { refetch(); }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-xs text-muted-foreground">最近 50 条请求记录（5 秒自动刷新）</p>
        <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={load}>
          刷新
        </Button>
      </div>
      <div className="rounded-xl border overflow-hidden">
        <Table>
          <TableHeader className="bg-muted/30">
            <TableRow>
              <TableHead>状态</TableHead>
              <TableHead>方法</TableHead>
              <TableHead>耗时</TableHead>
              <TableHead className="text-right">请求时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && logs.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="h-24 text-center">
                  <Loader2 className="h-4 w-4 animate-spin mx-auto opacity-20" />
                </TableCell>
              </TableRow>
            ) : logsError ? (
              <TableRow>
                <TableCell colSpan={4} className="h-24 text-center text-destructive italic">
                  加载失败
                </TableCell>
              </TableRow>
            ) : logs.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="h-24 text-center text-muted-foreground italic">
                  暂无记录
                </TableCell>
              </TableRow>
            ) : (
              logs.map((log) => (
                <TableRow key={log.id} className="text-xs group hover:bg-muted/30">
                  <TableCell>
                    <Badge
                      variant={log.status === "success" ? "default" : "destructive"}
                      className="h-4 text-[9px] uppercase"
                    >
                      {log.response_status || log.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono text-muted-foreground uppercase">
                    {log.request_method}
                  </TableCell>
                  <TableCell className="font-mono text-muted-foreground">
                    {log.duration_ms}ms
                  </TableCell>
                  <TableCell className="text-right text-muted-foreground opacity-60">
                    {new Date(log.created_at * 1000).toLocaleTimeString()}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
