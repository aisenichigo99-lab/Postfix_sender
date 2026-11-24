
import type { Job } from "./jobs";

export interface RelayConfig {
  host: string;
  port: number;
  tls_mode: "starttls" | "implicit";
  username: string;
  password: string;
}

export interface InstallationRequest {
  mail_name: string;
  configuration_type: "Internet with smarthost";
  relay: RelayConfig;
}

/**
 * Lance une installation de Postfix via l'API Go.
 */
export async function createInstallationJob(
  payload: InstallationRequest,
): Promise<Job> {
  const response = await fetch("/api/installation", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "same-origin",
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error(
      `Impossible de lancer l'installation (HTTP ${response.status}).`,
    );
  }

  const job: Job = await response.json();

  if (!job.id) {
    throw new Error("Réponse invalide du serveur : identifiant du job absent.");
  }

  return job;
}
