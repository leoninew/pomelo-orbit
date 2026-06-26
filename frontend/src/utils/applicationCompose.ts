export interface ApplicationEnvRow {
  key: string;
  value: string;
}

export interface ApplicationVolumeRow {
  hostPath: string;
  containerPath: string;
}

export interface ImageComposeInput {
  image: string;
  containerPort: number;
  envVars: ApplicationEnvRow[];
  volumes: ApplicationVolumeRow[];
}

export const imageCreateServiceName = 'app';

const envKeyPattern = /^[A-Za-z_][A-Za-z0-9_]*$/;

export function isValidEnvKey(key: string) {
  return envKeyPattern.test(key.trim());
}

export function normalizeEnvRows(rows: ApplicationEnvRow[]) {
  return rows
    .map((row) => ({ key: row.key.trim(), value: row.value.trim() }))
    .filter((row) => row.key || row.value);
}

export function normalizeVolumeRows(rows: ApplicationVolumeRow[]) {
  return rows
    .map((row) => ({ hostPath: row.hostPath.trim(), containerPath: row.containerPath.trim() }))
    .filter((row) => row.hostPath || row.containerPath);
}

export function buildEnvFile(rows: ApplicationEnvRow[]) {
  const envRows = normalizeEnvRows(rows);
  if (envRows.length === 0) {
    return '';
  }
  return envRows.map((row) => `${row.key}=${row.value}`).join('\n') + '\n';
}

export function buildImageCompose(input: ImageComposeInput) {
  const envRows = normalizeEnvRows(input.envVars);
  const volumeRows = normalizeVolumeRows(input.volumes);
  const lines = [
    'services:',
    `  ${imageCreateServiceName}:`,
    `    image: ${quoteYamlString(input.image.trim())}`,
    '    ports:',
    `      - ${quoteYamlString(`${input.containerPort}:${input.containerPort}`)}`,
  ];

  if (envRows.length > 0) {
    lines.push('    env_file:', '      - .env');
  }

  if (volumeRows.length > 0) {
    lines.push('    volumes:');
    for (const row of volumeRows) {
      lines.push(`      - ${quoteYamlString(`${row.hostPath}:${row.containerPath}`)}`);
    }
  }

  return lines.join('\n') + '\n';
}

function quoteYamlString(value: string) {
  return `"${value.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`;
}
