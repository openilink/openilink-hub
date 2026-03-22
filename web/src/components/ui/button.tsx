import { cn } from "../../lib/utils";
import { type ButtonHTMLAttributes } from "react";

type Variant = "default" | "secondary" | "destructive" | "ghost" | "outline";

const variants: Record<Variant, string> = {
  default: "bg-[var(--primary)] text-[var(--primary-foreground)] hover:opacity-90",
  secondary: "bg-[var(--secondary)] text-[var(--secondary-foreground)] hover:opacity-80",
  destructive: "bg-[var(--destructive)] text-white hover:opacity-90",
  ghost: "hover:bg-[var(--secondary)]",
  outline: "border border-[var(--border)] hover:bg-[var(--secondary)]",
};

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: "sm" | "md" | "lg";
}

export function Button({ className, variant = "default", size = "md", ...props }: ButtonProps) {
  const sizeClass = { sm: "px-3 py-1.5 text-xs", md: "px-4 py-2 text-sm", lg: "px-6 py-3 text-base" }[size];
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center rounded-[var(--radius)] font-medium transition-colors",
        "disabled:opacity-50 disabled:pointer-events-none cursor-pointer",
        variants[variant], sizeClass, className
      )}
      {...props}
    />
  );
}
