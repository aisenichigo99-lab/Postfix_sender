
import { useEffect, useState } from "react";

import {
  Server,
  RefreshCw,
  CheckCircle2,
  XCircle,
  ShieldCheck,
  Download,
  AlertTriangle,
} from "lucide-react";

import { getSystemInfo, type SystemInfo } from "../services/system";

export default function Installation() {
  const [system, setSystem] = useState<SystemInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");



const [hostname, setHostname] = useState("mail.example.com");
const [domain, setDomain] = useState("example.com");
const [mailName, setMailName] = useState("example.com");
const [installing, setInstalling] = useState(false);
const [progress, setProgress] = useState(0);
const [logs, setLogs] = useState<string[]>([]);


  async function detectSystem() {
    setLoading(true);
    setError("");

    try {
      const data = await getSystemInfo();
      setSystem(data);
    } catch {
      setError("Impossible de contacter le backend Go.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    detectSystem();
  }, []);

  return (
    <div className="space-y-7">

      <div>
        <h2 className="text-xl font-semibold text-white">
          Installation
        </h2>
        <p className="mt-1 text-sm text-slate-500">
          Prepare your Linux server for Postfix.
        </p>
      </div>

      <section className="rounded-xl border border-white/[0.06] bg-[#171e2b] p-6">

        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-blue-500/10 p-3 text-blue-400">
              <Server size={22} />
            </div>

            <div>
              <h3 className="font-semibold text-white">
                System detection
              </h3>
              <p className="text-xs text-slate-500">
                Operating system information
              </p>
            </div>
          </div>

          <button
            onClick={detectSystem}
            disabled={loading}
            className="flex items-center gap-2 rounded-lg border border-white/10 px-3 py-2 text-xs text-slate-300 hover:bg-white/5 disabled:opacity-50"
          >
            <RefreshCw
              size={14}
              className={loading ? "animate-spin" : ""}
            />
            Refresh
          </button>
        </div>

        {loading && (
          <p className="mt-6 text-sm text-slate-400">
            Detecting system...
          </p>
        )}

        {error && (
          <div className="mt-6 flex items-center gap-2 rounded-lg bg-red-500/10 p-4 text-sm text-red-400">
            <XCircle size={18} />
            {error}
          </div>
        )}

        {system && !loading && (
          <div className="mt-7 space-y-5">

            <div>
              <p className="text-xs text-slate-500">
                Distribution
              </p>
              <p className="mt-1 text-sm font-medium text-white">
                {system.pretty_name || system.distribution}
              </p>
            </div>

            <div className="grid grid-cols-2 gap-5 sm:grid-cols-4">
              <div>
                <p className="text-xs text-slate-500">Version</p>
                <p className="mt-1 text-sm text-white">
                  {system.version || "Unknown"}
                </p>
              </div>

              <div>
                <p className="text-xs text-slate-500">Architecture</p>
                <p className="mt-1 text-sm text-white">
                  {system.architecture}
                </p>
              </div>

              <div>
                <p className="text-xs text-slate-500">
                  Package manager
                </p>
                <p className="mt-1 text-sm text-white">
                  {system.package_manager}
                </p>
              </div>

              <div>
                <p className="text-xs text-slate-500">
                  Root privileges
                </p>
                <p className={`mt-1 text-sm ${
                  system.is_root
                    ? "text-emerald-400"
                    : "text-orange-400"
                }`}>
                  {system.is_root ? "Available" : "Not available"}
                </p>
              </div>
            </div>

            <div className="border-t border-white/[0.06] pt-5">
              <div className="flex items-center gap-2">
                {system.supported ? (
                  <CheckCircle2
                    size={18}
                    className="text-emerald-400"
                  />
                ) : (
                  <AlertTriangle
                    size={18}
                    className="text-orange-400"
                  />
                )}

                <span className="text-sm text-slate-300">
                  {system.supported
                    ? "Operating system supported"
                    : "Operating system not yet supported"}
                </span>
              </div>
            </div>

            <div className="flex items-center justify-between rounded-lg bg-[#101722] p-4">
              <div>
                <p className="text-sm font-medium text-white">
                  Postfix
                </p>
                <p className="mt-1 text-xs text-slate-500">
                  Service status: {system.postfix_status}
                </p>
              </div>

              <span className={`rounded-full px-3 py-1 text-xs ${
                system.postfix_installed
                  ? "bg-emerald-500/10 text-emerald-400"
                  : "bg-orange-500/10 text-orange-400"
              }`}>
                {system.postfix_installed
                  ? "Installed"
                  : "Not installed"}
              </span>
            </div>

            <div className="rounded-lg border border-blue-500/10 bg-blue-500/5 p-4">
              <div className="flex items-center gap-2 text-blue-400">
                <ShieldCheck size={18} />
                <span className="text-sm font-medium">
                  System verification
                </span>
              </div>

              <p className="mt-2 text-xs leading-5 text-slate-400">
                {system.supported
                  ? "Your distribution is supported by the current installation module."
                  : "The current installation module does not support this distribution yet."}
              </p>
            </div>

            {!system.postfix_installed && system.supported && (
              <button
                disabled={!system.is_root}
                className="flex w-full items-center justify-center gap-2 rounded-lg bg-blue-500 px-4 py-3 text-sm font-medium text-white hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-40"
              >
                <Download size={17} />
                Installation engine coming next
              </button>
            )}

          </div>
        )}
      </section>

<section className="rounded-xl border border-white/[0.06] bg-[#171e2b] p-6 space-y-5">

  <div>
    <h3 className="font-semibold text-white">
      Postfix configuration
    </h3>
    <p className="mt-1 text-xs text-slate-500">
      Define your mail server identity.
    </p>
  </div>

  <div className="space-y-2">
    <label className="text-xs text-slate-400">
      Mail hostname
    </label>
    <input
      value={hostname}
      onChange={(e) => setHostname(e.target.value)}
      placeholder="mail.example.com"
      className="w-full rounded-lg border border-white/10 bg-[#101722] px-4 py-3 text-sm text-white outline-none focus:border-blue-500"
    />
  </div>

  <div className="space-y-2">
    <label className="text-xs text-slate-400">
      Domain
    </label>
    <input
      value={domain}
      onChange={(e) => setDomain(e.target.value)}
      placeholder="example.com"
      className="w-full rounded-lg border border-white/10 bg-[#101722] px-4 py-3 text-sm text-white outline-none focus:border-blue-500"
    />
  </div>

  <div className="space-y-2">
    <label className="text-xs text-slate-400">
      Mail name
    </label>
    <input
      value={mailName}
      onChange={(e) => setMailName(e.target.value)}
      placeholder="example.com"
      className="w-full rounded-lg border border-white/10 bg-[#101722] px-4 py-3 text-sm text-white outline-none focus:border-blue-500"
    />
  </div>

  <button
    disabled={installing || !system?.supported || !system?.is_root}
    className="w-full rounded-lg bg-blue-500 px-4 py-3 text-sm font-medium text-white disabled:opacity-40"
  >
    Install Postfix
  </button>

</section>

    </div>
  );
}
