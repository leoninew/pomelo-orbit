/**
 * docker compose ps --format json 解析。
 * Compose v2 可能输出 JSON 数组、单对象，或每行一个对象（NDJSON）。
 */

export type ComposeContainer = {
  id: string;
  name: string;
  service: string;
  state: string;
  status: string;
  health: string;
  image?: string;
};

type ComposePsRow = {
  ID?: string;
  Id?: string;
  id?: string;
  Name?: string;
  name?: string;
  Service?: string;
  service?: string;
  State?: string;
  state?: string;
  Status?: string;
  status?: string;
  Health?: string;
  health?: string;
  Image?: string;
  image?: string;
};

function normalizeComposePsRow(row: ComposePsRow): ComposeContainer {
  return {
    id: String(row.ID ?? row.Id ?? row.id ?? ''),
    name: String(row.Name ?? row.name ?? ''),
    service: String(row.Service ?? row.service ?? ''),
    state: String(row.State ?? row.state ?? '').toLowerCase(),
    status: String(row.Status ?? row.status ?? ''),
    health: String(row.Health ?? row.health ?? ''),
    image: row.Image || row.image ? String(row.Image ?? row.image) : undefined,
  };
}

export function parseComposePsOutput(raw: string | undefined | null): ComposeContainer[] {
  const text = (raw ?? '').trim();
  if (!text) {
    return [];
  }

  try {
    const parsed: unknown = JSON.parse(text);
    if (Array.isArray(parsed)) {
      return parsed.map((item) => normalizeComposePsRow(item as ComposePsRow));
    }
    if (parsed && typeof parsed === 'object') {
      return [normalizeComposePsRow(parsed as ComposePsRow)];
    }
  } catch {
    // fall through to NDJSON
  }

  const items: ComposeContainer[] = [];
  for (const line of text.split('\n')) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }
    try {
      items.push(normalizeComposePsRow(JSON.parse(trimmed) as ComposePsRow));
    } catch {
      // skip malformed lines
    }
  }
  return items;
}
