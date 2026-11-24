
export interface Job {
  id: string;
  type: string;
  status: string;
  progress: number;
  created_at: string;
  updated_at: string;
}

export interface JobEvent {
  time: string;
  status: string;
  progress: number;
  message: string;
}

export async function createTestJob(): Promise<Job> {
  const response = await fetch("/api/test-job", {
    method: "POST",
  });

  if (!response.ok) {
    throw new Error("Unable to create test job");
  }

  return response.json();
}

export function watchJob(
  id: string,
  onEvent: (event: JobEvent) => void,
): EventSource {
  const source = new EventSource(`/api/jobs/${id}/events`);

  source.addEventListener("progress", (event: MessageEvent) => {
    const data: JobEvent = JSON.parse(event.data);
    onEvent(data);
  });

  return source;
}
