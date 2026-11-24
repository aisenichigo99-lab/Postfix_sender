
import {
  Server,
  Mail,
  Send,
  AlertCircle,
  CheckCircle2,
  Clock,
  ArrowUpRight,
} from "lucide-react";

import StatCard from "../components/StatCard";

const activities = [
  {
    title: "SMTP relay configuration",
    description: "Configuration updated successfully",
    time: "10 minutes ago",
    status: "success",
  },
  {
    title: "Postfix service",
    description: "Service started successfully",
    time: "25 minutes ago",
    status: "success",
  },
  {
    title: "SMTP connection",
    description: "Connection test completed",
    time: "1 hour ago",
    status: "success",
  },
];

export default function Dashboard() {
  return (
    <div className="space-y-8">

      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-white">
            Welcome back
          </h2>
          <p className="mt-1 text-sm text-slate-500">
            Here is what's happening with your infrastructure.
          </p>
        </div>

        <button className="flex items-center gap-2 rounded-lg bg-blue-500 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-blue-600">
          <Send size={16} />
          Send test email
        </button>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          title="Postfix Status"
          value="Unknown"
          description="Waiting for system connection"
          icon={Server}
        />

        <StatCard
          title="SMTP Relays"
          value="0"
          description="Configured providers"
          icon={Mail}
        />

        <StatCard
          title="Messages Sent"
          value="--"
          description="Awaiting monitoring data"
          icon={Send}
        />

        <StatCard
          title="Failed Messages"
          value="--"
          description="Awaiting monitoring data"
          icon={AlertCircle}
        />
      </div>

      <div className="grid grid-cols-1 gap-5 xl:grid-cols-3">

        <div className="space-y-5 xl:col-span-2">

          <section className="rounded-xl border border-white/[0.06] bg-[#171e2b] p-6">
            <div className="flex items-center justify-between">
              <h3 className="font-semibold text-white">
                Infrastructure status
              </h3>

              <span className="text-xs text-slate-500">
                System overview
              </span>
            </div>

            <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="rounded-lg bg-[#101722] p-4">
                <div className="flex items-center gap-2">
                  <Server size={18} className="text-blue-400" />
                  <span className="text-sm text-slate-300">
                    Postfix service
                  </span>
                </div>

                <p className="mt-3 text-lg font-medium text-slate-300">
                  Not connected
                </p>

                <p className="mt-1 text-xs text-slate-500">
                  System agent required
                </p>
              </div>

              <div className="rounded-lg bg-[#101722] p-4">
                <div className="flex items-center gap-2">
                  <Mail size={18} className="text-blue-400" />
                  <span className="text-sm text-slate-300">
                    SMTP Relay
                  </span>
                </div>

                <p className="mt-3 text-lg font-medium text-slate-300">
                  Not configured
                </p>

                <p className="mt-1 text-xs text-slate-500">
                  No active provider
                </p>
              </div>
            </div>
          </section>

          <section className="rounded-xl border border-white/[0.06] bg-[#171e2b] p-6">
            <div className="flex items-center justify-between">
              <h3 className="font-semibold text-white">
                Recent activity
              </h3>

              <button className="flex items-center gap-1 text-xs text-blue-400">
                View all <ArrowUpRight size={14} />
              </button>
            </div>

            <div className="mt-5 space-y-5">
              {activities.map((activity) => (
                <div
                  key={activity.title}
                  className="flex items-start gap-3"
                >
                  <CheckCircle2
                    size={18}
                    className="mt-0.5 text-emerald-400"
                  />

                  <div className="flex-1">
                    <p className="text-sm text-slate-200">
                      {activity.title}
                    </p>
                    <p className="mt-1 text-xs text-slate-500">
                      {activity.description}
                    </p>
                  </div>

                  <span className="text-[11px] text-slate-500">
                    {activity.time}
                  </span>
                </div>
              ))}
            </div>
          </section>
        </div>

        <section className="rounded-xl border border-white/[0.06] bg-[#171e2b] p-6">
          <h3 className="font-semibold text-white">
            Quick actions
          </h3>

          <div className="mt-5 space-y-3">
            {[
              {
                icon: Server,
                title: "Install Postfix",
                description: "Set up your mail server",
              },
              {
                icon: Mail,
                title: "Add SMTP Relay",
                description: "Connect an external provider",
              },
              {
                icon: Clock,
                title: "View Mail Queue",
                description: "Inspect pending messages",
              },
              {
                icon: AlertCircle,
                title: "Security Audit",
                description: "Check your configuration",
              },
            ].map((item) => {
              const Icon = item.icon;

              return (
                <button
                  key={item.title}
                  className="flex w-full items-center gap-3 rounded-lg bg-[#101722] p-3 text-left transition hover:bg-[#202b3c]"
                >
                  <div className="rounded-lg bg-blue-500/10 p-2 text-blue-400">
                    <Icon size={18} />
                  </div>

                  <div>
                    <p className="text-sm text-slate-200">
                      {item.title}
                    </p>
                    <p className="mt-1 text-xs text-slate-500">
                      {item.description}
                    </p>
                  </div>

                  <ArrowUpRight
                    size={15}
                    className="ml-auto text-slate-600"
                  />
                </button>
              );
            })}
          </div>
        </section>
      </div>
    </div>
  );
}
