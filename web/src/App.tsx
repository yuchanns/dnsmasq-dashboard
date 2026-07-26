import {
  AlertTriangle,
  ArrowDown,
  ArrowUp,
  Check,
  Clock3,
  Copy,
  EthernetPort,
  Gauge,
  Moon,
  RefreshCw,
  Router,
  Search,
  ShieldAlert,
  Sun,
  Wifi,
  WifiOff,
  X,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import type { Lease, LeaseStatus } from "./types";
import { useSnapshot } from "./useSnapshot";

type Filter = "all" | LeaseStatus;
type SortKey = "ip" | "device" | "status" | "expires";
type SortDirection = "asc" | "desc";

const statusOrder: Record<LeaseStatus, number> = {
  online: 0,
  recent: 1,
  offline: 2,
  conflict: 3,
};

function App() {
  const { snapshot, connection, receivedAt, error, refresh } = useSnapshot();
  const [filter, setFilter] = useState<Filter>("all");
  const [query, setQuery] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("ip");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");
  const [now, setNow] = useState(() => Date.now());
  const [theme, setTheme] = useState<"light" | "dark">(() => {
    const saved = localStorage.getItem("leaseboard-theme");
    if (saved === "light" || saved === "dark") return saved;
    return window.matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark";
  });

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("leaseboard-theme", theme);
  }, [theme]);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1_000);
    return () => window.clearInterval(timer);
  }, []);

  const leases = useMemo(() => {
    if (!snapshot) return [];
    const normalized = query.trim().toLowerCase();
    const result = snapshot.leases.filter((lease) => {
      if (filter !== "all" && lease.status !== filter) return false;
      if (!normalized) return true;
      return [lease.hostname, lease.ip, lease.mac, lease.clientId]
        .join(" ")
        .toLowerCase()
        .includes(normalized);
    });

    return result.sort((left, right) => {
      let value = 0;
      switch (sortKey) {
        case "ip":
          value = compareIP(left.ip, right.ip);
          break;
        case "device":
          value = displayName(left).localeCompare(displayName(right));
          break;
        case "status":
          value = statusOrder[left.status] - statusOrder[right.status];
          break;
        case "expires":
          value = expiryValue(left) - expiryValue(right);
          break;
      }
      return sortDirection === "asc" ? value : -value;
    });
  }, [filter, query, snapshot, sortDirection, sortKey]);

  const setSort = (next: SortKey) => {
    if (sortKey === next) {
      setSortDirection((value) => (value === "asc" ? "desc" : "asc"));
      return;
    }
    setSortKey(next);
    setSortDirection("asc");
  };

  const summary = snapshot?.summary;
  const utilization = summary ? Math.round((summary.leased / summary.capacity) * 100) : 0;

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand">
          <span className="brand-mark" aria-hidden="true">
            <Router size={19} strokeWidth={2.2} />
          </span>
          <div>
            <h1>Leaseboard</h1>
            <p>{summary?.name ?? "Local network"}</p>
          </div>
        </div>

        <div className="topbar-actions">
          <div className={`live-state live-state--${connection}`}>
            <span className="live-state__dot" />
            {connection === "live" ? "Live" : connection === "connecting" ? "Reconnecting" : "Offline"}
          </div>
          <button
            className="icon-button"
            type="button"
            title="Refresh snapshot"
            aria-label="Refresh snapshot"
            onClick={() => void refresh()}
          >
            <RefreshCw size={17} />
          </button>
          <button
            className="icon-button"
            type="button"
            title={`Use ${theme === "dark" ? "light" : "dark"} theme`}
            aria-label={`Use ${theme === "dark" ? "light" : "dark"} theme`}
            onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
          >
            {theme === "dark" ? <Sun size={17} /> : <Moon size={17} />}
          </button>
        </div>
      </header>

      <main>
        <section className="network-overview" aria-label="Network overview">
          <div className="overview-heading">
            <div>
              <span className="eyebrow">DHCP pool</span>
              <strong>
                {summary?.poolStart ?? "—"} <span>to</span> {summary?.poolEnd ?? "—"}
              </strong>
            </div>
            <div className="interface-label">
              <EthernetPort size={15} />
              {summary?.interface ?? "—"}
            </div>
          </div>

          <div className="metrics">
            <Metric
              icon={<Gauge size={18} />}
              label="Leased"
              value={summary?.leased ?? 0}
              detail={`${utilization}% of pool`}
              tone="cyan"
            />
            <Metric
              icon={<Wifi size={18} />}
              label="Reachable"
              value={summary?.online ?? 0}
              detail={`${summary?.recent ?? 0} recently seen`}
              tone="green"
            />
            <Metric
              icon={<EthernetPort size={18} />}
              label="Available"
              value={summary?.available ?? 0}
              detail={`${summary?.capacity ?? 0} total addresses`}
              tone="amber"
            />
            <Metric
              icon={<ShieldAlert size={18} />}
              label="Conflicts"
              value={summary?.conflicts ?? 0}
              detail={summary?.conflicts ? "Needs attention" : "No address conflicts"}
              tone="red"
            />
          </div>

          <div className="pool-bar" aria-label={`${utilization}% of DHCP pool leased`}>
            <span style={{ width: `${Math.min(100, utilization)}%` }} />
          </div>
        </section>

        {error && (
          <div className="warning-banner" role="status">
            <AlertTriangle size={17} />
            <span>{error}</span>
          </div>
        )}

        <section className="leases-section">
          <div className="section-heading">
            <div>
              <h2>Leases</h2>
              <p>
                {leases.length} shown
                {receivedAt ? ` · updated ${relativeTime(receivedAt.getTime(), now)}` : ""}
              </p>
            </div>

            <div className="table-controls">
              <div className="search-field">
                <Search size={16} aria-hidden="true" />
                <input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder="Search device, IP or MAC"
                  aria-label="Search leases"
                />
                {query && (
                  <button type="button" onClick={() => setQuery("")} aria-label="Clear search">
                    <X size={15} />
                  </button>
                )}
              </div>
              <div className="filter-group" aria-label="Filter leases">
                {(["all", "online", "recent", "offline", "conflict"] as const).map((value) => (
                  <button
                    type="button"
                    className={filter === value ? "is-active" : ""}
                    onClick={() => setFilter(value)}
                    key={value}
                  >
                    {filterLabel(value)}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <div className="lease-table">
            <div className="lease-row lease-row--header" role="row">
              <SortButton label="Status" field="status" current={sortKey} direction={sortDirection} onClick={setSort} />
              <SortButton label="Device" field="device" current={sortKey} direction={sortDirection} onClick={setSort} />
              <SortButton label="IP address" field="ip" current={sortKey} direction={sortDirection} onClick={setSort} />
              <span>MAC address</span>
              <SortButton label="Lease" field="expires" current={sortKey} direction={sortDirection} onClick={setSort} />
              <span className="sr-only">Actions</span>
            </div>

            {leases.map((lease) => (
              <LeaseRow lease={lease} now={now} key={`${lease.ip}-${lease.mac}`} />
            ))}

            {leases.length === 0 && (
              <div className="empty-state">
                <Search size={21} />
                <strong>No matching leases</strong>
                <span>Adjust the search or status filter.</span>
              </div>
            )}
          </div>
        </section>
      </main>

      <footer>
        <span>Leaseboard</span>
        <span>{snapshot?.healthy ? "Source healthy" : "Source degraded"}</span>
      </footer>
    </div>
  );
}

function Metric({
  icon,
  label,
  value,
  detail,
  tone,
}: {
  icon: React.ReactNode;
  label: string;
  value: number;
  detail: string;
  tone: "cyan" | "green" | "amber" | "red";
}) {
  return (
    <div className={`metric metric--${tone}`}>
      <span className="metric__icon">{icon}</span>
      <div>
        <span className="metric__label">{label}</span>
        <strong>{value}</strong>
        <small>{detail}</small>
      </div>
    </div>
  );
}

function SortButton({
  label,
  field,
  current,
  direction,
  onClick,
}: {
  label: string;
  field: SortKey;
  current: SortKey;
  direction: SortDirection;
  onClick: (field: SortKey) => void;
}) {
  const active = current === field;
  return (
    <button type="button" className={active ? "sort-button is-active" : "sort-button"} onClick={() => onClick(field)}>
      {label}
      {active && (direction === "asc" ? <ArrowUp size={13} /> : <ArrowDown size={13} />)}
    </button>
  );
}

function LeaseRow({ lease, now }: { lease: Lease; now: number }) {
  const [copied, setCopied] = useState(false);
  const copyIP = async () => {
    await navigator.clipboard.writeText(lease.ip);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1_500);
  };

  return (
    <div className="lease-row" role="row">
      <div className="status-cell">
        <span className={`status-dot status-dot--${lease.status}`} />
        <div>
          <strong>{statusLabel(lease.status)}</strong>
          <span>{neighborLabel(lease)}</span>
        </div>
      </div>
      <div className="device-cell">
        <span className={`device-icon device-icon--${lease.status}`} aria-hidden="true">
          {lease.status === "offline" ? <WifiOff size={17} /> : <Wifi size={17} />}
        </span>
        <div>
          <strong>{displayName(lease)}</strong>
          <span className="client-id">{lease.clientId || "No client identifier"}</span>
          <span className={`mobile-status mobile-status--${lease.status}`}>
            <i />
            {statusLabel(lease.status)} · {neighborLabel(lease)}
          </span>
        </div>
      </div>
      <code className="ip-cell">{lease.ip}</code>
      <code className="mac-cell">{lease.mac}</code>
      <div className="expiry-cell">
        <Clock3 size={15} />
        <div>
          <strong>{formatExpiry(lease, now)}</strong>
          <span>{lease.infinite ? "Static lease" : formatAbsolute(lease.expiresAt)}</span>
        </div>
      </div>
      <button className="copy-button" type="button" title="Copy IP address" aria-label={`Copy ${lease.ip}`} onClick={() => void copyIP()}>
        {copied ? <Check size={15} /> : <Copy size={15} />}
      </button>
    </div>
  );
}

function filterLabel(value: Filter) {
  return value === "all" ? "All" : statusLabel(value);
}

function statusLabel(status: LeaseStatus) {
  return {
    online: "Online",
    recent: "Recent",
    offline: "Offline",
    conflict: "Conflict",
  }[status];
}

function neighborLabel(lease: Lease) {
  if (lease.status === "conflict") return "MAC mismatch";
  return lease.neighborState ? lease.neighborState.toLowerCase() : "not observed";
}

function displayName(lease: Lease) {
  return lease.hostname || `Device ${lease.ip.split(".").at(-1)}`;
}

function expiryValue(lease: Lease) {
  if (lease.infinite || !lease.expiresAt) return Number.MAX_SAFE_INTEGER;
  return new Date(lease.expiresAt).getTime();
}

function formatExpiry(lease: Lease, now: number) {
  if (lease.infinite || !lease.expiresAt) return "Never";
  const seconds = Math.max(0, Math.floor((new Date(lease.expiresAt).getTime() - now) / 1_000));
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h ${remainingMinutes}m`;
}

function formatAbsolute(value: string | null) {
  if (!value) return "";
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function relativeTime(value: number, now: number) {
  const seconds = Math.max(0, Math.floor((now - value) / 1_000));
  if (seconds < 5) return "just now";
  if (seconds < 60) return `${seconds}s ago`;
  return `${Math.floor(seconds / 60)}m ago`;
}

function compareIP(left: string, right: string) {
  const toNumber = (value: string) =>
    value.split(".").reduce((total, part) => total * 256 + Number(part), 0);
  return toNumber(left) - toNumber(right);
}

export default App;
