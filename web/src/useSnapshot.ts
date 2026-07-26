import { useCallback, useEffect, useRef, useState } from "react";
import type { ConnectionState, Snapshot } from "./types";

interface SnapshotState {
  snapshot: Snapshot | null;
  connection: ConnectionState;
  receivedAt: Date | null;
  error: string | null;
  refresh: () => Promise<void>;
}

export function useSnapshot(): SnapshotState {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [connection, setConnection] = useState<ConnectionState>("connecting");
  const [receivedAt, setReceivedAt] = useState<Date | null>(null);
  const [error, setError] = useState<string | null>(null);
  const sourceRef = useRef<EventSource | null>(null);

  const applySnapshot = useCallback((value: Snapshot) => {
    setSnapshot(value);
    setReceivedAt(new Date());
    setError(value.healthy ? null : value.warnings.join(" · "));
  }, []);

  const refresh = useCallback(async () => {
    const response = await fetch("/api/v1/snapshot", {
      cache: "no-store",
      headers: { Accept: "application/json" },
    });
    if (!response.ok) {
      throw new Error(`Snapshot request failed with ${response.status}`);
    }
    applySnapshot((await response.json()) as Snapshot);
  }, [applySnapshot]);

  useEffect(() => {
    let cancelled = false;
    refresh().catch((reason: unknown) => {
      if (!cancelled) {
        setError(reason instanceof Error ? reason.message : "Unable to load leases");
      }
    });

    const source = new EventSource("/api/v1/events");
    sourceRef.current = source;
    source.onopen = () => {
      if (!cancelled) {
        setConnection("live");
      }
    };
    source.addEventListener("snapshot", (event) => {
      if (cancelled) return;
      applySnapshot(JSON.parse((event as MessageEvent<string>).data) as Snapshot);
    });
    source.onerror = () => {
      if (!cancelled) {
        setConnection(source.readyState === EventSource.CONNECTING ? "connecting" : "offline");
      }
    };

    return () => {
      cancelled = true;
      source.close();
      sourceRef.current = null;
    };
  }, [applySnapshot, refresh]);

  return { snapshot, connection, receivedAt, error, refresh };
}
