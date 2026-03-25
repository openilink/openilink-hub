import { Blocks } from "lucide-react";

export function AppIcon({ icon, iconUrl, size = "h-12 w-12" }: {
  icon?: string;
  iconUrl?: string;
  size?: string;
}) {
  if (iconUrl) return <img src={iconUrl} alt="" className={`${size} rounded-xl object-cover border`} />;
  if (icon) return <div className={`${size} rounded-xl bg-muted flex items-center justify-center text-lg border`}>{icon}</div>;
  return <div className={`${size} rounded-xl bg-muted flex items-center justify-center border`}><Blocks className="h-5 w-5 text-muted-foreground/40" /></div>;
}
