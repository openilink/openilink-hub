import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Bot,
  MessageSquare,
  Zap,
  Plus,
  ArrowRight,
  Wifi,
  Workflow,
  Cpu,
  AlertTriangle,
  TrendingUp,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import { Skeleton } from "@/components/ui/skeleton";

export function DashboardOverviewPage() {
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    api.stats()
      .then(setStats)
      .finally(() => setLoading(false));
  }, []);

  if (loading) return (
    <div className="space-y-8">
      <div className="flex justify-between items-center"><Skeleton className="h-10 w-48" /><Skeleton className="h-10 w-32" /></div>
      <div className="grid gap-4 md:grid-cols-4"><Skeleton className="h-24 w-full rounded-2xl" /><Skeleton className="h-24 w-full rounded-2xl" /><Skeleton className="h-24 w-full rounded-2xl" /><Skeleton className="h-24 w-full rounded-2xl" /></div>
      <div className="grid gap-4 md:grid-cols-7"><Skeleton className="h-96 col-span-4 rounded-[2rem]" /><Skeleton className="h-96 col-span-3 rounded-[2rem]" /></div>
    </div>
  );

  return (
    <div className="space-y-10">
      {/* Hero Section */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div className="space-y-1">
          <h2 className="text-4xl font-black tracking-tighter">概览</h2>
          <p className="text-muted-foreground font-medium">查看账号状态和消息统计。</p>
        </div>
        <div className="flex items-center gap-3">
          <Button variant="outline" className="rounded-full h-12 px-6 font-bold text-xs uppercase tracking-widest border-border/50 bg-background/50 hover:bg-muted" onClick={() => navigate("/dashboard/accounts")}>
            账号管理
          </Button>
          <Button className="rounded-full h-12 px-8 shadow-2xl shadow-primary/30 gap-2 font-bold text-xs uppercase tracking-widest transition-all hover:scale-105 active:scale-95" onClick={() => navigate("/dashboard/accounts")}>
            <Plus className="h-4 w-4" /> 添加账号
          </Button>
        </div>
      </div>

      {/* Metrics */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        {[
          { label: "在线账号", value: stats?.online_bots || 0, sub: `共 ${stats?.total_bots} 个`, icon: Bot, color: "text-blue-500" },
          { label: "消息总量", value: stats?.total_messages || 0, sub: "", icon: MessageSquare, color: "text-green-500" },
          { label: "转发规则", value: stats?.total_channels || 0, sub: "", icon: Workflow, color: "text-purple-500" },
          { label: "WebSocket 连接", value: stats?.connected_ws || 0, sub: "", icon: Wifi, color: "text-orange-500" },
        ].map((m, i) => (
          <Card key={i} className="border-border/50 bg-card/50 shadow-sm relative overflow-hidden group hover:border-primary/30 transition-all cursor-default">
            <div className="absolute -right-4 -bottom-4 opacity-[0.03] group-hover:opacity-[0.08] transition-all rotate-12 group-hover:rotate-0">
               <m.icon className="h-32 w-32" />
            </div>
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardTitle className="text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground/70">{m.label}</CardTitle>
                <div className={`h-6 w-6 rounded-lg bg-muted flex items-center justify-center ${m.color}`}>
                   <m.icon className="h-3.5 w-3.5" />
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-black tabular-nums tracking-tighter">{m.value.toLocaleString()}</div>
              {m.sub && (
                <div className="flex items-center gap-2 mt-1">
                   <p className="text-[10px] text-muted-foreground font-bold uppercase tracking-tighter">{m.sub}</p>
                   <div className="h-px flex-1 bg-gradient-to-r from-border to-transparent" />
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-8 md:grid-cols-2 lg:grid-cols-7">
        {/* Quick Start Roadmap */}
        <Card className="col-span-4 border-border/50 bg-card/30 rounded-[2.5rem] overflow-hidden shadow-xl">
          <CardHeader className="px-10 py-8 border-b border-border/50 bg-muted/20">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <CardTitle className="text-2xl font-black tracking-tight">快速开始</CardTitle>
                <CardDescription className="text-sm font-medium">三步开始使用</CardDescription>
              </div>
              <div className="h-12 w-12 rounded-2xl bg-primary/10 flex items-center justify-center text-primary">
                 <TrendingUp className="h-6 w-6" />
              </div>
            </div>
          </CardHeader>
          <CardContent className="p-0">
            <div className="divide-y divide-border/50">
               {[
                 { step: "01", title: "添加微信账号", desc: "扫码登录你的微信，连接到平台。", link: "/dashboard/accounts", icon: Cpu },
                 { step: "02", title: "创建转发规则", desc: "设置消息转发到你的服务器或 AI。", link: "/dashboard/accounts", icon: Workflow },
                 { step: "03", title: "安装应用", desc: "从市场安装现成的扩展功能。", link: "/dashboard/apps", icon: Zap },
               ].map((item, i) => (
                 <Link key={i} to={item.link} className="flex items-center gap-8 p-10 hover:bg-primary/[0.02] transition-all group">
                    <div className="text-4xl font-black text-primary/10 group-hover:text-primary transition-colors italic tracking-tighter">{item.step}</div>
                    <div className="h-14 w-14 rounded-2xl bg-background border border-border/50 shadow-inner flex items-center justify-center shrink-0 group-hover:scale-110 transition-transform">
                       <item.icon className="h-7 w-7 text-muted-foreground group-hover:text-primary transition-colors" />
                    </div>
                    <div className="flex-1">
                       <h4 className="font-bold text-xl tracking-tight">{item.title}</h4>
                       <p className="text-sm text-muted-foreground mt-1 max-w-md">{item.desc}</p>
                    </div>
                    <div className="h-10 w-10 rounded-full bg-muted flex items-center justify-center opacity-0 group-hover:opacity-100 transition-all translate-x-4 group-hover:translate-x-0">
                       <ArrowRight className="h-5 w-5 text-primary" />
                    </div>
                 </Link>
               ))}
            </div>
          </CardContent>
        </Card>

        {/* Sidebar */}
        <div className="col-span-3 space-y-8">
           {stats?.online_bots === 0 && (
             <div className="p-5 rounded-3xl border-destructive/20 bg-destructive/5 flex gap-4 animate-in shake-in duration-500">
                <div className="h-10 w-10 rounded-full bg-destructive/10 flex items-center justify-center shrink-0">
                   <AlertTriangle className="h-5 w-5 text-destructive" />
                </div>
                <div className="space-y-1">
                   <p className="text-xs font-black text-destructive uppercase tracking-widest">暂无在线账号</p>
                   <p className="text-[11px] text-destructive/80 leading-relaxed font-medium">还没有在线的微信账号，请先添加一个。</p>
                </div>
             </div>
           )}
        </div>
      </div>
    </div>
  );
}
