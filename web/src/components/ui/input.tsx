import { cn } from "../../lib/utils";
import { type InputHTMLAttributes } from "react";

export function Input({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={cn(
        "w-full rounded-[var(--radius)] border border-[var(--border)] bg-[var(--background)]",
        "px-3 py-2 text-sm text-[var(--foreground)] placeholder:text-[var(--muted-foreground)]",
        "focus:outline-none focus:border-[var(--primary)] transition-colors",
        className
      )}
      {...props}
    />
  );
}
