import type { GatewayCreateReq, GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';

export type GatewayConfigFormMode = 'create' | 'edit';
export type GatewayAcmeProfile = '' | 'http' | 'dns' | 'http-dns';

export interface GatewayConfigForm {
  name: string;
  code: string;
  traefik_component_name: string;
  rest_api_url: string;
  rest_ready_timeout_seconds: string;
  base_domain: string;
  initial_component_image: string;
  initial_component_pull_policy: string;
  default_entrypoint: string;
  tls_mode: string;
  acme_profile: GatewayAcmeProfile;
  acme_email: string;
  dns_api_token: string;
}

export type GatewayConfigFormErrors = Record<string, string>;

export function emptyGatewayConfigForm(): GatewayConfigForm {
  return {
    name: 'Traefik',
    code: 'traefik',
    traefik_component_name: 'traefik',
    rest_api_url: 'http://localhost:8080',
    rest_ready_timeout_seconds: '20',
    base_domain: 'lvh.me',
    initial_component_image: 'traefik:3.6',
    initial_component_pull_policy: 'missing',
    default_entrypoint: 'web',
    tls_mode: 'none',
    acme_profile: '',
    acme_email: '',
    dns_api_token: '',
  };
}

export function gatewayConfigFormFromResponse(value: GatewayResp): GatewayConfigForm {
  return {
    name: value.name,
    code: value.code,
    traefik_component_name: value.traefik_component_name,
    rest_api_url: value.rest_api_url,
    rest_ready_timeout_seconds: String(value.rest_ready_timeout_seconds),
    base_domain: value.base_domain,
    initial_component_image: '',
    initial_component_pull_policy: 'missing',
    default_entrypoint: value.default_entrypoint,
    tls_mode: value.tls_mode,
    acme_profile: isGatewayAcmeProfile(value.acme_profile) ? value.acme_profile : '',
    acme_email: value.acme_email,
    dns_api_token: value.dns_api_token,
  };
}

function isGatewayAcmeProfile(value: string): value is GatewayAcmeProfile {
  return value === '' || value === 'http' || value === 'dns' || value === 'http-dns';
}

function profileUsesHTTP(profile: GatewayAcmeProfile): boolean {
  return profile === 'http' || profile === 'http-dns';
}

function profileUsesDNS(profile: GatewayAcmeProfile): boolean {
  return profile === 'dns' || profile === 'http-dns';
}

function isValidURL(value: string): boolean {
  try {
    const url = new URL(value);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}

function isValidBaseDomain(value: string): boolean {
  return /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$/i.test(
    value
  );
}

function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
}

function isValidTimeout(value: string): boolean {
  const parsed = Number(value);
  return Number.isInteger(parsed) && parsed >= 1 && parsed <= 300;
}

export function validateGatewayConfigForm(
  form: GatewayConfigForm,
  mode: GatewayConfigFormMode
): GatewayConfigFormErrors {
  const errors: GatewayConfigFormErrors = {};
  if (!form.name.trim()) errors.name = 'nameRequired';
  if (mode === 'create') {
    if (!/^[a-z][a-z0-9-]*$/.test(form.code.trim())) errors.code = 'codeInvalid';
    if (!form.initial_component_image.trim()) errors.initial_component_image = 'imageRequired';
    if (!['always', 'missing', 'never'].includes(form.initial_component_pull_policy)) {
      errors.initial_component_pull_policy = 'pullPolicyInvalid';
    }
  }
  if (!/^[a-z][a-z0-9-]*$/.test(form.traefik_component_name.trim())) {
    errors.traefik_component_name = 'componentNameInvalid';
  }
  if (!isValidURL(form.rest_api_url.trim())) errors.rest_api_url = 'restApiUrlInvalid';
  if (!isValidBaseDomain(form.base_domain.trim())) errors.base_domain = 'baseDomainInvalid';
  if (!isValidTimeout(form.rest_ready_timeout_seconds)) {
    errors.rest_ready_timeout_seconds = 'restReadyTimeoutInvalid';
  }
  if (!['web', 'websecure'].includes(form.default_entrypoint)) {
    errors.default_entrypoint = 'defaultEntrypointInvalid';
  }
  if (!['none', 'tls', 'letsencrypt'].includes(form.tls_mode)) errors.tls_mode = 'tlsModeInvalid';
  if (!isGatewayAcmeProfile(form.acme_profile)) errors.acme_profile = 'acmeProfileInvalid';
  if (form.acme_profile !== '' && !isValidEmail(form.acme_email.trim())) {
    errors.acme_email = 'acmeEmailInvalid';
  }
  if (profileUsesDNS(form.acme_profile) && !form.dns_api_token.trim()) {
    errors.dns_api_token = 'dnsApiTokenRequired';
  }
  if (form.tls_mode === 'letsencrypt' && !profileUsesHTTP(form.acme_profile)) {
    errors.tls_mode = 'tlsModeRequiresHttpProfile';
  }
  return errors;
}

function configRequest(form: GatewayConfigForm) {
  return {
    name: form.name.trim(),
    traefik_component_name: form.traefik_component_name.trim(),
    rest_api_url: form.rest_api_url.trim(),
    rest_ready_timeout_seconds: Number(form.rest_ready_timeout_seconds),
    base_domain: form.base_domain.trim(),
    default_entrypoint: form.default_entrypoint,
    tls_mode: form.tls_mode,
    acme_profile: form.acme_profile,
    acme_email: form.acme_profile === '' ? '' : form.acme_email.trim(),
    dns_api_token: profileUsesDNS(form.acme_profile) ? form.dns_api_token.trim() : '',
  };
}

export function gatewayCreateRequestFromForm(
  form: GatewayConfigForm,
  projectID: string
): GatewayCreateReq {
  return {
    project_id: projectID,
    code: form.code.trim(),
    initial_component_image: form.initial_component_image.trim(),
    initial_component_pull_policy: form.initial_component_pull_policy,
    ...configRequest(form),
  };
}
