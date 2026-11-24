
import { Bell, Search, CircleUserRound } from "lucide-react";

export default function Header() {
  return (
    <header className="flex h-[76px] items-center justify-between border-b border-white/[0.06] bg-[#10141d] px-8">
      <div>
        <h2 className="text-lg font-semibold text-white">
          Dashboard
        </h2>
        <p className="text-xs text-slate-500">
          Overview of your mail infrastructure
        </p>
      </div>

      <div className="flex items-center gap-4">
        <button className="rounded-lg p-2 text-slate-400 hover:bg-white/[0.05]">
          <Search size={19} />
        </button>

        <button className="relative rounded-lg p-2 text-slate-400 hover:bg-white/[0.05]">
          <Bell size={19} />
          <span className="absolute right-2 top-2 h-1.5 w-1.5 rounded-full bg-blue-400" />
        </button>

        <div className="flex items-center gap-2 border-l border-white/[0.08] pl-4">
          <CircleUserRound size={23} className="text-slate-400" />
          <span className="text-sm text-slate-300">
            Administrator
          </span>
        </div>
      </div>
    </header>
  );
}
