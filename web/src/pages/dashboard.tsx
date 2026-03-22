import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Bot, Cable, MessageSquare, Wifi } from "lucide-react";
import { Card } from "../components/ui/card";
import { api } from "../lib/api";

export function DashboardPage() {
  const [stats, setStats] = useState<any>(null);

  useEffect(() => {
    api.stats().then(setStats).catch(() => {});
  }, []);

  const items = [
    { label: "Bot 总数", value: stats?.total_bots ?? "-", icon: Bot, href: "/bots" },
    { label: "在线 Bot", value: stats?.online_bots ?? "-", icon: Wifi },
    { label: "分发通道", value: stats?.total_sublevels ?? "-", icon: Cable, href: "/sublevels" },
    { label: "消息总数", value: stats?.total_messages ?? "-", icon: MessageSquare },
  ];

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold">概览</h2>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {items.map((item) => {
          const content = (
            <Card key={item.label} className="flex items-center gap-4 hover:border-[var(--primary)] transition-colors">
              <item.icon className="w-8 h-8 text-[var(--muted-foreground)]" />
              <div>
                <p className="text-2xl font-bold">{item.value}</p>
                <p className="text-xs text-[var(--muted-foreground)]">{item.label}</p>
              </div>
            </Card>
          );
          return item.href ? <Link to={item.href} key={item.label}>{content}</Link> : <div key={item.label}>{content}</div>;
        })}
      </div>
    </div>
  );
}
