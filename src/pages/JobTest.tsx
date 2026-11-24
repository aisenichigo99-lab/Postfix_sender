
import { useEffect, useState } from "react";
import { createTestJob, watchJob } from "../services/jobs";
import type { JobEvent } from "../services/jobs";

export default function JobTest() {
  const [jobId, setJobId] = useState("");
  const [progress, setProgress] = useState(0);
  const [status, setStatus] = useState("idle");
  const [logs, setLogs] = useState<JobEvent[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!jobId) return;

    const source = watchJob(jobId, (event) => {
      setProgress(event.progress);
      setStatus(event.status);
      setLogs((previous) => [...previous, event]);

      if (["succeeded", "failed", "canceled"].includes(event.status)) {
        source.close();
      }
    });

    return () => source.close();
  }, [jobId]);

  async function startTest() {
    setLoading(true);

    try {
      setLogs([]);
      setProgress(0);
      setStatus("queued");

      const job = await createTestJob();
      setJobId(job.id);
    } catch (error) {
      setStatus("error");
      console.error(error);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Job Manager Test</h1>
        <p className="mt-1 text-slate-400">
          Integration test for background tasks and live logs.
        </p>
      </div>

      <button
        onClick={startTest}
        disabled={loading || status === "running"}
        className="rounded-lg bg-blue-500 px-4 py-2 text-white disabled:opacity-50"
      >
        {loading ? "Starting..." : "Run integration test"}
      </button>

      <div className="rounded-xl border border-white/10 bg-[#171E2B] p-5">
        <div className="mb-3 flex justify-between">
          <span>Progress</span>
          <span>{progress}%</span>
        </div>

        <div className="h-2 overflow-hidden rounded-full bg-slate-700">
          <div
            className="h-full rounded-full bg-blue-500 transition-all"
            style={{ width: `${progress}%` }}
          />
        </div>

        <p className="mt-3 text-sm text-slate-400">
          Status: {status}
        </p>
      </div>

      <div className="rounded-xl border border-white/10 bg-[#171E2B] p-5">
        <h2 className="mb-4 font-semibold">Live logs</h2>

        <div className="max-h-80 space-y-2 overflow-y-auto font-mono text-sm">
          {logs.map((log, index) => (
            <div key={index} className="text-slate-300">
              <span className="mr-3 text-slate-500">
                {new Date(log.time).toLocaleTimeString()}
              </span>
              {log.message}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
