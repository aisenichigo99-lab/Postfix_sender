
import { NavLink } from "react-router-dom";
import {
  LayoutDashboard,
  Download,
  Settings,
  Server,
  ShieldCheck,
  Activity,
  Network,
  Mail,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";

interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
}

const navigation = [
  {
    title: "GENERAL",
    items: [
      { label: "Dashboard", icon: LayoutDashboard, path: "/" },
    ],
  },
  {
    title: "CONFIGURATION",
    items: [
      { label: "Installation", icon: Download, path: "/installation" },
      { label: "Postfix", icon: Server, path: "#" },
      { label: "SMTP Relays", icon: Network, path: "#" },
      { label: "Security", icon: ShieldCheck, path: "#" },
    ],
  },
  {
    title: "MONITORING",
    items: [
      { label: "Activity", icon: Activity, path: "#" },
      { label: "Mail Queue", icon: Mail, path: "#" },
    ],
  },
];

export default function Sidebar({
  collapsed,
  onToggle,
}: SidebarProps) {
  return (
    <aside
      className={`fixed left-0 top-0 z-30 flex h-screen flex-col border-r border-white/[0.06] bg-[#131925] transition-all duration-300 ${
        collapsed ? "w-[76px]" : "w-[250px]"
      }`}
    >
      <div className="flex h-[76px] items-center gap-3 border-b border-white/[0.06] px-5">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-blue-500/15 text-blue-400">
          <Mail size={21} />
        </div>

        {!collapsed && (
          <div>
            <h1 className="text-sm font-semibold text-white">
              Postfix Manager
            </h1>
            <p className="text-[10px] text-slate-500">
              SMTP Infrastructure
            </p>
          </div>
        )}
      </div>

      <nav className="flex-1 space-y-6 overflow-y-auto p-3">
        {navigation.map((group) => (
          <div key={group.title}>
            {!collapsed && (
              <p className="mb-2 px-3 text-[10px] font-semibold tracking-wider text-slate-600">
                {group.title}
              </p>
            )}

            <div className="space-y-1">
              {group.items.map((item) => {
                const Icon = item.icon;

                return (
                  <NavLink
                    key={item.label}
                    to={item.path}
                    title={collapsed ? item.label : undefined}
                    className={({ isActive }) =>
                      `flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition ${
                        isActive && item.path !== "#"
                          ? "bg-blue-500/10 text-blue-400"
                          : "text-slate-400 hover:bg-white/[0.04] hover:text-white"
                      }`
                    }
                  >
                    <Icon size={18} />
                    {!collapsed && <span>{item.label}</span>}
                  </NavLink>
                );
              })}
            </div>
          </div>
        ))}
      </nav>

      <div className="border-t border-white/[0.06] p-3">
        <button
          onClick={onToggle}
          className="flex w-full items-center justify-center rounded-lg p-2 text-slate-500 hover:bg-white/[0.05] hover:text-white"
        >
          {collapsed ? (
            <ChevronRight size={18} />
          ) : (
            <ChevronLeft size={18} />
          )}
        </button>
      </div>
    </aside>
  );
}
