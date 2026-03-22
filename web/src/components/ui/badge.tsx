import { cn } from "../../lib/utils";

const variants = {
  default: "bg-[var(--primary)] text-[var(--primary-foreground)]",
  secondary: "bg-[var(--secondary)] text-[var(--secondary-foreground)]",
  destructive: "bg-[var(--destructive)] text-white",
  outline: "border border-[var(--border)] text-[var(--muted-foreground)]",
};

export function Badge({ variant = "default", className, children }: {
  variant?: keyof typeof variants; className?: string; children: React.ReactNode;
}) {
  return (
    <span className={cn("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium", variants[variant], className)}>
      {children}
    </span>
  );
}
