import { describe, expect, it } from 'vitest';
import { emptyGatewayConfigForm, validateGatewayConfigForm } from './gatewayConfigForm';

describe('gatewayConfigForm', () => {
  it('requires an email for a selected ACME profile', () => {
    const form = emptyGatewayConfigForm();
    form.acme_profile = 'http';

    expect(validateGatewayConfigForm(form, 'create')).toMatchObject({
      acme_email: 'acmeEmailInvalid',
    });

    form.acme_email = 'ops@example.com';
    expect(validateGatewayConfigForm(form, 'create').acme_email).toBeUndefined();
  });

  it('requires a Gateway DNS token for DNS profiles', () => {
    const form = emptyGatewayConfigForm();
    form.acme_profile = 'dns';
    form.acme_email = 'ops@example.com';

    expect(validateGatewayConfigForm(form, 'create')).toMatchObject({
      dns_api_token: 'dnsApiTokenRequired',
    });

    form.dns_api_token = 'cfat_gateway_token';
    expect(validateGatewayConfigForm(form, 'create').dns_api_token).toBeUndefined();
  });

  it('requires an HTTP-capable profile for the letsencrypt TLS mode', () => {
    const form = emptyGatewayConfigForm();
    form.tls_mode = 'letsencrypt';
    form.acme_profile = 'dns';
    form.acme_email = 'ops@example.com';
    form.dns_api_token = 'cfat_gateway_token';

    expect(validateGatewayConfigForm(form, 'create')).toMatchObject({
      tls_mode: 'tlsModeRequiresHttpProfile',
    });

    form.acme_profile = 'http-dns';
    expect(validateGatewayConfigForm(form, 'create').tls_mode).toBeUndefined();
  });
});
