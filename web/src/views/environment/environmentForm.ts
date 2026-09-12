import type {
  EnvironmentLocalTargetResp,
  EnvironmentResp,
  ProjectEnvironmentUpdateReq,
} from '@/gen/proto/orbit/v1/environment/environment';
import type { ProjectInitializationEnvironmentReq } from '@/gen/proto/orbit/v1/project_initialization/project_initialization';
import {
  defaultDeploymentWorkspaceRoot,
  isLocalWorkspaceRoot,
  isPlatformWorkspaceRoot,
} from '@/utils/deploymentEnvironment';

export type EnvironmentTargetType = 'local' | 'ssh';
export type EnvironmentPlatform = 'linux' | 'windows';

export type EnvironmentForm = {
  state: string;
  targetType: EnvironmentTargetType;
  platform: EnvironmentPlatform;
  host: string;
  port: number;
  username: string;
  workspaceRoot: string;
};

export type EnvironmentFormErrors = {
  host: string;
  port: string;
  username: string;
  workspaceRoot: string;
};

export type EnvironmentTargetSnapshot = {
  target_type?: string;
  state?: string;
  workspace_root?: string;
  local?: Pick<EnvironmentLocalTargetResp, 'workspace_root' | 'platform' | 'host' | 'username'>;
  ssh?: {
    platform?: string;
    host?: string;
    port?: number;
    username?: string;
    workspace_root?: string;
  };
};

export type EnvironmentFormMessages = Pick<
  EnvironmentFormErrors,
  'host' | 'port' | 'username' | 'workspaceRoot'
>;

export function emptyEnvironmentForm(): EnvironmentForm {
  return {
    state: 'active',
    targetType: 'local',
    platform: 'linux',
    host: '',
    port: 22,
    username: '',
    workspaceRoot: '',
  };
}

export function emptyEnvironmentFormErrors(): EnvironmentFormErrors {
  return { host: '', port: '', username: '', workspaceRoot: '' };
}

export function hydrateEnvironmentForm(
  snapshot: EnvironmentTargetSnapshot | undefined,
  localWorkspaceRoot = ''
): EnvironmentForm {
  const form = emptyEnvironmentForm();
  if (!snapshot) {
    form.workspaceRoot = localWorkspaceRoot;
    return form;
  }
  form.state = snapshot.state || 'active';
  if (snapshot.target_type === 'ssh' && snapshot.ssh) {
    form.targetType = 'ssh';
    form.platform = snapshot.ssh.platform === 'windows' ? 'windows' : 'linux';
    form.host = snapshot.ssh.host || '';
    form.port = snapshot.ssh.port || 22;
    form.username = snapshot.ssh.username || '';
    form.workspaceRoot = snapshot.ssh.workspace_root || defaultDeploymentWorkspaceRoot();
    return form;
  }
  form.targetType = 'local';
  form.platform = snapshot.local?.platform === 'windows' ? 'windows' : 'linux';
  form.workspaceRoot =
    snapshot.local?.workspace_root || snapshot.workspace_root || localWorkspaceRoot;
  return form;
}

export function environmentWorkspacePlaceholder(
  form: EnvironmentForm,
  localWorkspaceRoot: string
): string {
  return form.targetType === 'local' ? localWorkspaceRoot : defaultDeploymentWorkspaceRoot();
}

export function applyEnvironmentTargetType(
  form: EnvironmentForm,
  targetType: EnvironmentTargetType,
  localDisplay: EnvironmentLocalTargetResp | undefined,
  localWorkspaceRoot: string
): EnvironmentForm {
  const next = { ...form, targetType };
  if (targetType === 'ssh') {
    if (
      form.targetType !== 'ssh' &&
      (localDisplay?.platform === 'linux' || localDisplay?.platform === 'windows')
    ) {
      next.platform = localDisplay.platform;
    }
    if (!isPlatformWorkspaceRoot(next.platform, next.workspaceRoot)) {
      next.workspaceRoot = defaultDeploymentWorkspaceRoot();
    }
    if (localDisplay && next.platform === localDisplay.platform) {
      next.host = next.host || localDisplay.host || '';
      next.username = next.username || localDisplay.username || '';
    }
    return next;
  }
  if (!isLocalWorkspaceRoot(next.workspaceRoot)) {
    next.workspaceRoot = localWorkspaceRoot;
  }
  return next;
}

export function applyEnvironmentPlatform(
  form: EnvironmentForm,
  platform: EnvironmentPlatform,
  localDisplay: EnvironmentLocalTargetResp | undefined
): EnvironmentForm {
  const next = { ...form, platform };
  if (!isPlatformWorkspaceRoot(platform, next.workspaceRoot)) {
    next.workspaceRoot = defaultDeploymentWorkspaceRoot();
  }
  if (form.targetType === 'ssh' && localDisplay && platform === localDisplay.platform) {
    next.host = next.host || localDisplay.host || '';
    next.username = next.username || localDisplay.username || '';
  }
  return next;
}

export function validateEnvironmentForm(
  form: EnvironmentForm,
  messages: EnvironmentFormMessages
): EnvironmentFormErrors {
  const errors = emptyEnvironmentFormErrors();
  if (form.targetType === 'local') {
    errors.workspaceRoot = isLocalWorkspaceRoot(form.workspaceRoot) ? '' : messages.workspaceRoot;
    return errors;
  }
  errors.host = form.host.trim() && !/\s/.test(form.host) ? '' : messages.host;
  errors.port =
    Number.isInteger(form.port) && form.port >= 1 && form.port <= 65535 ? '' : messages.port;
  errors.username = form.username.trim() && !/[\r\n]/.test(form.username) ? '' : messages.username;
  errors.workspaceRoot = isPlatformWorkspaceRoot(form.platform, form.workspaceRoot)
    ? ''
    : messages.workspaceRoot;
  return errors;
}

export function environmentFormHasErrors(errors: EnvironmentFormErrors): boolean {
  return Object.values(errors).some(Boolean);
}

export function assignEnvironmentFormErrors(
  target: EnvironmentFormErrors,
  next: EnvironmentFormErrors
): void {
  target.host = next.host;
  target.port = next.port;
  target.username = next.username;
  target.workspaceRoot = next.workspaceRoot;
}

export function environmentTargetPayload(form: EnvironmentForm): {
  target_type: EnvironmentTargetType;
  local?: { workspace_root: string };
  ssh?: {
    platform: EnvironmentPlatform;
    host: string;
    port: number;
    username: string;
    workspace_root: string;
  };
} {
  if (form.targetType === 'local') {
    return { target_type: 'local', local: { workspace_root: form.workspaceRoot.trim() } };
  }
  return {
    target_type: 'ssh',
    ssh: {
      platform: form.platform,
      host: form.host.trim(),
      port: form.port,
      username: form.username.trim(),
      workspace_root: form.workspaceRoot.trim(),
    },
  };
}

export function environmentUpdateRequestFromForm(
  form: EnvironmentForm
): ProjectEnvironmentUpdateReq {
  const payload = environmentTargetPayload(form);
  return { state: form.state, ...payload };
}

export function initializationEnvironmentRequestFromForm(
  form: EnvironmentForm
): ProjectInitializationEnvironmentReq {
  return environmentTargetPayload(form);
}

export function environmentFormDirty(
  form: EnvironmentForm,
  snapshot: EnvironmentTargetSnapshot | undefined
): boolean {
  if (!snapshot) {
    return true;
  }
  if (form.state && snapshot.state && form.state !== snapshot.state) {
    return true;
  }
  if (form.targetType !== snapshot.target_type) {
    return true;
  }
  if (form.targetType === 'local') {
    return (
      form.workspaceRoot.trim() !== (snapshot.local?.workspace_root || snapshot.workspace_root)
    );
  }
  const ssh = snapshot.ssh;
  return (
    !ssh ||
    form.platform !== ssh.platform ||
    form.host.trim() !== ssh.host ||
    form.port !== ssh.port ||
    form.username.trim() !== ssh.username ||
    form.workspaceRoot.trim() !== ssh.workspace_root
  );
}

export function environmentFormFromResponse(environment: EnvironmentResp): EnvironmentForm {
  return hydrateEnvironmentForm(environment);
}
