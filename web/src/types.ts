export type LeaseStatus = "online" | "recent" | "offline" | "conflict";

export interface Lease {
  ip: string;
  mac: string;
  hostname: string;
  clientId: string;
  expiresAt: string | null;
  infinite: boolean;
  status: LeaseStatus;
  neighborState: string;
  observedMac?: string;
}

export interface NetworkSummary {
  name: string;
  interface: string;
  poolStart: string;
  poolEnd: string;
  capacity: number;
  leased: number;
  available: number;
  online: number;
  recent: number;
  conflicts: number;
}

export interface Snapshot {
  revision: string;
  generatedAt: string;
  healthy: boolean;
  summary: NetworkSummary;
  leases: Lease[];
  warnings: string[];
}

export type ConnectionState = "connecting" | "live" | "offline";
