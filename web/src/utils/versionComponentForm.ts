export interface EnvFormRow {
  key: string;
  value: string;
}

export interface PortFormRow {
  host_port: number;
  container_port: number;
}

export interface MountFormRow {
  source_type: string;
  source: string;
  target: string;
  read_only: boolean;
  content: string;
  content_mode: string;
}

const fileMountSuffixes = [
  '.json',
  '.yml',
  '.yaml',
  '.toml',
  '.pem',
  '.key',
  '.crt',
  '.conf',
  '.cfg',
  '.txt',
  '.env',
];

export function parseEnvJson(raw?: string): EnvFormRow[] {
  if (!raw?.trim()) {
    return [];
  }
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed
      .map((item) => {
        const row = item as { key?: string; value?: string };
        return { key: String(row.key || ''), value: String(row.value ?? '') };
      })
      .filter((row) => row.key || row.value);
  } catch {
    return [];
  }
}

export function serializeEnvRows(rows: EnvFormRow[]): string | undefined {
  const items = rows
    .map((row) => ({ key: row.key.trim(), value: row.value }))
    .filter((row) => row.key);
  if (items.length === 0) {
    return undefined;
  }
  return JSON.stringify(items);
}

export function isValidPort(port: number): boolean {
  return Number.isInteger(port) && port >= 1 && port <= 65535;
}

function isPortInRange(port: number): boolean {
  return port >= 1 && port <= 65535;
}

export function parsePortsJson(raw?: string): PortFormRow[] {
  if (!raw?.trim()) {
    return [];
  }
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) {
      return [];
    }
    const rows: PortFormRow[] = [];
    for (const item of parsed) {
      if (typeof item === 'number') {
        if (isPortInRange(item)) {
          rows.push({ host_port: item, container_port: item });
        }
        continue;
      }
      if (typeof item === 'string') {
        const text = item.trim();
        if (!text) {
          continue;
        }
        const parts = text.split(':');
        if (parts.length === 1) {
          const port = Number(parts[0]);
          if (isPortInRange(port)) {
            rows.push({ host_port: port, container_port: port });
          }
          continue;
        }
        const hostPort = Number(parts[parts.length - 2]);
        const containerPort = Number(parts[parts.length - 1]);
        if (isPortInRange(hostPort) && isPortInRange(containerPort)) {
          rows.push({ host_port: hostPort, container_port: containerPort });
        }
        continue;
      }
      if (item && typeof item === 'object') {
        const row = item as {
          host_port?: number;
          published?: number;
          target?: number;
          container_port?: number;
        };
        const hostPort = Number(row.host_port ?? row.published ?? 0);
        const containerPort = Number(row.container_port ?? row.target ?? 0);
        if (isPortInRange(hostPort) && isPortInRange(containerPort)) {
          rows.push({ host_port: hostPort, container_port: containerPort });
        }
      }
    }
    return rows;
  } catch {
    return [];
  }
}

export function serializePortsRows(rows: PortFormRow[]): string | undefined {
  const items = rows
    .filter((row) => isValidPort(Number(row.host_port)) && isValidPort(Number(row.container_port)))
    .map((row) => `${Number(row.host_port)}:${Number(row.container_port)}`);
  if (items.length === 0) {
    return undefined;
  }
  return JSON.stringify(items);
}

export function isFileMountPath(source: string, target: string): boolean {
  for (const path of [source, target]) {
    const lower = path.toLowerCase();
    if (fileMountSuffixes.some((suffix) => lower.endsWith(suffix))) {
      return true;
    }
  }
  return false;
}

export function isFileMountRow(row: MountFormRow): boolean {
  return row.source_type === 'logical' && isFileMountPath(row.source, row.target);
}

export function parseMountsJson(raw?: string): MountFormRow[] {
  if (!raw?.trim()) {
    return [];
  }
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed.map((item) => {
      const row = item as {
        source_type?: string;
        source?: string;
        target?: string;
        read_only?: boolean;
        content?: string;
        content_mode?: string;
      };
      return {
        source_type: String(row.source_type || 'logical'),
        source: String(row.source || ''),
        target: String(row.target || ''),
        read_only: Boolean(row.read_only),
        content: String(row.content ?? ''),
        content_mode: String(row.content_mode || 'seed') === 'sync' ? 'sync' : 'seed',
      };
    });
  } catch {
    return [];
  }
}

export function serializeMountRows(rows: MountFormRow[]): string | undefined {
  const items = rows
    .map((row) => {
      const item: {
        source_type: string;
        source: string;
        target: string;
        read_only: boolean;
        content?: string;
        content_mode?: string;
      } = {
        source_type: row.source_type.trim(),
        source: row.source.trim(),
        target: row.target.trim(),
        read_only: row.read_only,
      };
      if (
        item.source_type === 'logical' &&
        isFileMountPath(item.source, item.target) &&
        (row.content.trim() || row.content_mode === 'sync')
      ) {
        if (row.content) {
          item.content = row.content;
        }
        item.content_mode = row.content_mode === 'sync' ? 'sync' : 'seed';
      }
      return item;
    })
    .filter((row) => row.source_type && row.source && row.target);
  if (items.length === 0) {
    return undefined;
  }
  return JSON.stringify(items);
}
