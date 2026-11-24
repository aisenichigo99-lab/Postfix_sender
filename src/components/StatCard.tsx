
import type { LucideIcon } from "lucide-react";

interface StatCardProps {
  title: string;
  value: string;
  description: string;
  icon: LucideIcon;
}

export default function StatCard({
  title,
  value,
  description,
  icon: Icon,
}: StatCardProps) {
  return (
    <div className="rounded-xl border border-white/[0.06] bg-[#171e2b] p-5">
      <div className="flex items-center justify-between">
        <p className="text-sm text-slate-400">{title}</p>

        <div className="rounded-lg bg-blue-500/10 p-2 text-blue-400">
          <Icon size={19} />
        </div>
      </div>

      <h3 className="mt-5 text-3xl font-semibold text-white">
        {value}
      </h3>

      <p className="mt-2 text-xs text-slate-500">
        {description}
      </p>
    </div>
  );
}
