
export interface SystemInfo {
  distribution: string;
  version: string;
  pretty_name: string;
  architecture: string;
  package_manager: string;
  is_root: boolean;
  postfix_installed: boolean;
  postfix_status: string;
  supported: boolean;
}

export async function getSystemInfo(): Promise<SystemInfo> {
  const response = await fetch("/api/system");

  if (!response.ok) {
    throw new Error("Unable to retrieve system information");
  }

  return response.json();
}
