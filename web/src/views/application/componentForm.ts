import type {
  ComponentEnv,
  ComponentHealthcheck,
  ComponentMount,
  ComponentPort,
  ComponentResources,
  ComponentTmpfs,
  ComponentUlimit,
  VersionComponentAdvancedUpdateReq,
  VersionComponentBasicUpdateReq,
  VersionComponentCreateReq,
  VersionComponentDependenciesUpdateReq,
  VersionComponentEnvUpdateReq,
  VersionComponentMountsUpdateReq,
  VersionComponentPortsUpdateReq,
  VersionComponentReq,
  VersionComponentResp,
  VersionComponentRuntimeUpdateReq,
} from '@/gen/proto/orbit/v1/application/version';

export interface PortRow {
  host_port: string;
  container_port: string;
}

export interface MountRow {
  source_type: string;
  source: string;
  target: string;
  read_only: boolean;
  source_is_host_path: boolean;
  content: string;
  mode: string;
  ignore_if_exists: boolean;
}

export interface DependencyRow {
  name: string;
  condition: string;
}

export interface TmpfsRow {
  target: string;
  size_bytes: string;
  mode: string;
}

export interface UlimitRow {
  name: string;
  soft: string;
  hard: string;
}

export interface ComponentForm {
  name: string;
  image: string;
  command: string;
  env: ComponentEnv[];
  ports: PortRow[];
  mounts: MountRow[];
  dependencies: DependencyRow[];
  healthcheck_enabled: boolean;
  healthcheck_disabled: boolean;
  healthcheck_test_mode: string;
  healthcheck_test: string;
  healthcheck_interval: string;
  healthcheck_timeout: string;
  healthcheck_retries: string;
  healthcheck_start_period: string;
  healthcheck_start_interval: string;
  pull_policy: string;
  restart_policy: string;
  resources: {
    limit_cpus: string;
    limit_memory: string;
    reservation_cpus: string;
    reservation_memory: string;
  };
  tmpfs: TmpfsRow[];
  ulimits: UlimitRow[];
}

export type ComponentFormValidation<T = VersionComponentReq> =
  { valid: true; value: T } | { valid: false; error: ComponentFormError };

export type ComponentFormError =
  | 'nameImage'
  | 'componentName'
  | 'command'
  | 'env'
  | 'ports'
  | 'mounts'
  | 'dependencies'
  | 'healthcheck'
  | 'tmpfs'
  | 'ulimits';

function inputText(value: string | undefined): string {
  return value === undefined ? '' : value;
}

export function emptyComponentForm(): ComponentForm {
  return {
    name: '',
    image: '',
    command: '',
    env: [],
    ports: [],
    mounts: [],
    dependencies: [],
    healthcheck_enabled: false,
    healthcheck_disabled: false,
    healthcheck_test_mode: '',
    healthcheck_test: '',
    healthcheck_interval: '',
    healthcheck_timeout: '',
    healthcheck_retries: '',
    healthcheck_start_period: '',
    healthcheck_start_interval: '',
    pull_policy: 'missing',
    restart_policy: '',
    resources: {
      limit_cpus: '',
      limit_memory: '',
      reservation_cpus: '',
      reservation_memory: '',
    },
    tmpfs: [],
    ulimits: [],
  };
}

export function componentFormFromResponse(component: VersionComponentResp): ComponentForm {
  const healthcheck = component.healthcheck;
  const resources = component.resources;
  return {
    name: component.name,
    image: component.image,
    command: component.command,
    env: component.env.map((item) => ({ key: item.key, value: item.value })),
    ports: component.ports.map((item) => ({
      host_port: String(item.host_port),
      container_port: String(item.container_port),
    })),
    mounts: component.mounts.map((item) => ({
      source_type: item.source_type,
      source: item.source,
      target: item.target,
      read_only: item.read_only,
      source_is_host_path: item.source_is_host_path,
      content: inputText(item.content),
      mode: inputText(item.mode),
      ignore_if_exists: item.ignore_if_exists,
    })),
    dependencies: component.dependencies.map((item) => ({
      name: item.name,
      condition: item.condition,
    })),
    healthcheck_enabled: healthcheck !== undefined,
    healthcheck_disabled: healthcheck?.disabled === true,
    healthcheck_test_mode: inputText(healthcheck?.test_mode),
    healthcheck_test: inputText(healthcheck?.test),
    healthcheck_interval: inputText(healthcheck?.interval),
    healthcheck_timeout: inputText(healthcheck?.timeout),
    healthcheck_retries: healthcheck?.retries === undefined ? '' : String(healthcheck.retries),
    healthcheck_start_period: inputText(healthcheck?.start_period),
    healthcheck_start_interval: inputText(healthcheck?.start_interval),
    pull_policy: inputText(component.pull_policy),
    restart_policy: inputText(component.restart_policy),
    resources: {
      limit_cpus: inputText(resources?.limit_cpus),
      limit_memory: inputText(resources?.limit_memory),
      reservation_cpus: inputText(resources?.reservation_cpus),
      reservation_memory: inputText(resources?.reservation_memory),
    },
    tmpfs: component.tmpfs.map((item) => ({
      target: item.target,
      size_bytes: String(item.size_bytes),
      mode: item.mode,
    })),
    ulimits: component.ulimits.map((item) => ({
      name: item.name,
      soft: String(item.soft),
      hard: String(item.hard),
    })),
  };
}

function isInteger(value: string): boolean {
  return /^-?\d+$/.test(value) && Number.isSafeInteger(Number(value));
}

function optionalText(value: string): string | undefined {
  return value === '' ? undefined : value;
}

function optionalNonNegativeInteger(value: string): number | undefined | null {
  if (value === '') {
    return undefined;
  }
  if (!isInteger(value) || Number(value) < 0) {
    return null;
  }
  return Number(value);
}

function buildHealthcheck(form: ComponentForm): ComponentHealthcheck | undefined | null {
  if (!form.healthcheck_enabled) {
    return undefined;
  }
  if (!form.healthcheck_disabled) {
    if (
      (form.healthcheck_test_mode !== 'CMD' && form.healthcheck_test_mode !== 'CMD-SHELL') ||
      form.healthcheck_test.trim() === ''
    ) {
      return null;
    }
  }
  const retries = optionalNonNegativeInteger(form.healthcheck_retries);
  if (retries === null) {
    return null;
  }
  return {
    test_mode: form.healthcheck_test_mode,
    test: form.healthcheck_test,
    interval: optionalText(form.healthcheck_interval),
    timeout: optionalText(form.healthcheck_timeout),
    retries,
    start_period: optionalText(form.healthcheck_start_period),
    start_interval: optionalText(form.healthcheck_start_interval),
    disabled: form.healthcheck_disabled,
  };
}

function buildResources(resourceForm: ComponentForm['resources']): ComponentResources | undefined {
  const resources: ComponentResources = {
    limit_cpus: optionalText(resourceForm.limit_cpus),
    limit_memory: optionalText(resourceForm.limit_memory),
    reservation_cpus: optionalText(resourceForm.reservation_cpus),
    reservation_memory: optionalText(resourceForm.reservation_memory),
  };
  return Object.values(resources).some((value) => value !== undefined) ? resources : undefined;
}

function buildPorts(rows: PortRow[]): ComponentPort[] | null {
  const ports: ComponentPort[] = [];
  for (const row of rows) {
    if (!isInteger(row.host_port) || !isInteger(row.container_port)) {
      return null;
    }
    const hostPort = Number(row.host_port);
    const containerPort = Number(row.container_port);
    if (hostPort < 1 || hostPort > 65535 || containerPort < 1 || containerPort > 65535) {
      return null;
    }
    ports.push({ host_port: hostPort, container_port: containerPort });
  }
  return ports;
}

function buildMounts(rows: MountRow[]): ComponentMount[] | null {
  const mounts: ComponentMount[] = [];
  for (const row of rows) {
    if (row.source_type === '' || row.source === '' || row.target === '') {
      return null;
    }
    const contentOptionsAllowed = row.source_type === 'controlled_file';
    if (!contentOptionsAllowed && (row.content !== '' || row.mode !== '' || row.ignore_if_exists)) {
      return null;
    }
    if (row.source_is_host_path && row.source_type !== 'directory' && row.source_type !== 'file') {
      return null;
    }
    if (contentOptionsAllowed && !/^0[0-7]{3}$/.test(row.mode)) {
      return null;
    }
    mounts.push({
      source_type: row.source_type,
      source: row.source,
      target: row.target,
      read_only: row.read_only,
      source_is_host_path: row.source_is_host_path,
      content: optionalText(row.content),
      mode: row.mode,
      ignore_if_exists: row.ignore_if_exists,
    });
  }
  return mounts;
}

function buildTmpfs(rows: TmpfsRow[]): ComponentTmpfs[] | null {
  const tmpfs: ComponentTmpfs[] = [];
  for (const row of rows) {
    if (
      row.target === '' ||
      !isInteger(row.size_bytes) ||
      Number(row.size_bytes) < 1048576 ||
      Number(row.size_bytes) > 8589934592 ||
      !/^[0-7]{3,4}$/.test(row.mode)
    ) {
      return null;
    }
    tmpfs.push({ target: row.target, size_bytes: Number(row.size_bytes), mode: row.mode });
  }
  return tmpfs;
}

function buildUlimits(rows: UlimitRow[]): ComponentUlimit[] | null {
  const ulimits: ComponentUlimit[] = [];
  for (const row of rows) {
    if (row.name === '' || !isInteger(row.soft) || !isInteger(row.hard)) {
      return null;
    }
    ulimits.push({ name: row.name, soft: Number(row.soft), hard: Number(row.hard) });
  }
  return ulimits;
}

export function componentResourcesRequestFromForm(
  resources: ComponentForm['resources']
): ComponentResources | undefined {
  return buildResources(resources);
}

export function componentTmpfsRequestFromForm(
  tmpfsRows: TmpfsRow[]
): ComponentFormValidation<ComponentTmpfs[]> {
  const tmpfs = buildTmpfs(tmpfsRows);
  if (tmpfs === null) {
    return { valid: false, error: 'tmpfs' };
  }
  return { valid: true, value: tmpfs };
}

export function componentUlimitsRequestFromForm(
  ulimitRows: UlimitRow[]
): ComponentFormValidation<ComponentUlimit[]> {
  const ulimits = buildUlimits(ulimitRows);
  if (ulimits === null) {
    return { valid: false, error: 'ulimits' };
  }
  return { valid: true, value: ulimits };
}

export function componentRequestFromForm(form: ComponentForm): ComponentFormValidation {
  const basic = componentBasicRequestFromForm(form);
  if (!basic.valid) {
    return basic;
  }
  const runtime = componentRuntimeRequestFromForm(form);
  if (!runtime.valid) {
    return runtime;
  }
  const ports = componentPortsRequestFromForm(form);
  if (!ports.valid) {
    return ports;
  }
  const env = componentEnvRequestFromForm(form);
  if (!env.valid) {
    return env;
  }
  const mounts = componentMountsRequestFromForm(form);
  if (!mounts.valid) {
    return mounts;
  }
  const dependencies = componentDependenciesRequestFromForm(form);
  if (!dependencies.valid) {
    return dependencies;
  }
  const advanced = componentAdvancedRequestFromForm(form);
  if (!advanced.valid) {
    return advanced;
  }
  return {
    valid: true,
    value: {
      ...basic.value,
      ...runtime.value,
      ...ports.value,
      ...env.value,
      ...mounts.value,
      ...dependencies.value,
      ...advanced.value,
    },
  };
}

export function componentBasicRequestFromForm(
  form: ComponentForm
): ComponentFormValidation<VersionComponentBasicUpdateReq> {
  if (form.name === '' || form.image === '') {
    return { valid: false, error: 'nameImage' };
  }
  if (!/^[a-z][a-z0-9-]*$/.test(form.name)) {
    return { valid: false, error: 'componentName' };
  }
  return {
    valid: true,
    value: {
      name: form.name,
      image: form.image,
      command: form.command,
      pull_policy: optionalText(form.pull_policy),
      restart_policy: optionalText(form.restart_policy),
    },
  };
}

export function componentCreateRequestFromForm(
  form: ComponentForm
): ComponentFormValidation<VersionComponentCreateReq> {
  if (form.name === '' || form.image === '') {
    return { valid: false, error: 'nameImage' };
  }
  if (!/^[a-z][a-z0-9-]*$/.test(form.name)) {
    return { valid: false, error: 'componentName' };
  }
  return {
    valid: true,
    value: {
      name: form.name,
      image: form.image,
      command: form.command,
      pull_policy: optionalText(form.pull_policy),
      restart_policy: optionalText(form.restart_policy),
    },
  };
}

export function componentRuntimeRequestFromForm(
  form: ComponentForm
): ComponentFormValidation<VersionComponentRuntimeUpdateReq> {
  const healthcheck = buildHealthcheck(form);
  if (healthcheck === null) {
    return { valid: false, error: 'healthcheck' };
  }
  return {
    valid: true,
    value: {
      healthcheck,
    },
  };
}

export function componentPortsRequestFromForm(
  form: ComponentForm
): ComponentFormValidation<VersionComponentPortsUpdateReq> {
  const ports = buildPorts(form.ports);
  if (ports === null) {
    return { valid: false, error: 'ports' };
  }
  return { valid: true, value: { ports } };
}

export function componentEnvRequestFromForm(
  form: ComponentForm
): ComponentFormValidation<VersionComponentEnvUpdateReq> {
  if (form.env.some((item) => item.key === '')) {
    return { valid: false, error: 'env' };
  }
  return {
    valid: true,
    value: { env: form.env.map((item) => ({ key: item.key, value: item.value })) },
  };
}

export function componentMountsRequestFromForm(
  form: ComponentForm
): ComponentFormValidation<VersionComponentMountsUpdateReq> {
  const mounts = buildMounts(form.mounts);
  if (mounts === null) {
    return { valid: false, error: 'mounts' };
  }
  return { valid: true, value: { mounts } };
}

export function componentDependenciesRequestFromForm(
  form: ComponentForm
): ComponentFormValidation<VersionComponentDependenciesUpdateReq> {
  if (form.dependencies.some((item) => item.name === '' || item.condition === '')) {
    return { valid: false, error: 'dependencies' };
  }
  return {
    valid: true,
    value: {
      dependencies: form.dependencies.map((item) => ({
        name: item.name,
        condition: item.condition,
      })),
    },
  };
}

export function componentAdvancedRequestFromForm(
  form: ComponentForm
): ComponentFormValidation<VersionComponentAdvancedUpdateReq> {
  const tmpfs = componentTmpfsRequestFromForm(form.tmpfs);
  if (!tmpfs.valid) {
    return tmpfs;
  }
  const ulimits = componentUlimitsRequestFromForm(form.ulimits);
  if (!ulimits.valid) {
    return ulimits;
  }
  return {
    valid: true,
    value: {
      resources: componentResourcesRequestFromForm(form.resources),
      tmpfs: tmpfs.value,
      ulimits: ulimits.value,
    },
  };
}
