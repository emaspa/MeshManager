import clsx from "clsx";
import type { ButtonHTMLAttributes, ReactNode } from "react";

export function Button({
  className,
  variant = "default",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "default" | "primary" | "danger" | "ghost";
}) {
  return (
    <button
      className={clsx(
        "inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed",
        variant === "default" &&
          "bg-(--color-panel-2) border border-(--color-border) hover:bg-(--color-border)",
        variant === "primary" && "bg-(--color-accent) text-white hover:bg-(--color-accent-2)",
        variant === "danger" && "bg-(--color-bad) text-white hover:opacity-90",
        variant === "ghost" && "hover:bg-(--color-panel-2)",
        className,
      )}
      {...props}
    />
  );
}

export function Card({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={clsx(
        "rounded-lg border border-(--color-border) bg-(--color-panel) p-4",
        className,
      )}
    >
      {children}
    </div>
  );
}

export function Badge({
  children,
  tone = "muted",
}: {
  children: ReactNode;
  tone?: "good" | "bad" | "warn" | "muted";
}) {
  const map = {
    good: "bg-(--color-good)/15 text-(--color-good)",
    bad: "bg-(--color-bad)/15 text-(--color-bad)",
    warn: "bg-(--color-warn)/15 text-(--color-warn)",
    muted: "bg-(--color-muted)/15 text-(--color-muted)",
  };
  return (
    <span className={clsx("rounded px-2 py-0.5 text-xs font-medium", map[tone])}>{children}</span>
  );
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="flex flex-col gap-1 text-sm">
      <span className="text-(--color-muted)">{label}</span>
      {children}
    </label>
  );
}

export function Input(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className={clsx(
        "rounded-md border border-(--color-border) bg-(--color-bg) px-3 py-1.5 text-sm outline-none focus:border-(--color-accent)",
        props.className,
      )}
    />
  );
}

export function Spinner() {
  return (
    <div className="h-4 w-4 animate-spin rounded-full border-2 border-(--color-border) border-t-(--color-accent)" />
  );
}
