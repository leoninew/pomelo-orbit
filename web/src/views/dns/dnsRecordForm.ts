export interface DNSRecordForm {
  content: string;
  ttl: string;
  proxied: boolean;
}

export type DNSRecordErrors = Partial<Record<'content' | 'ttl', 'invalidIPv4' | 'invalidTTL'>>;

export function validateDNSRecordForm(form: DNSRecordForm): DNSRecordErrors {
  const errors: DNSRecordErrors = {};
  const octets = form.content.trim().split('.');
  if (
    octets.length !== 4 ||
    octets.some((part) => !/^(0|[1-9]\d{0,2})$/.test(part) || Number(part) > 255)
  ) {
    errors.content = 'invalidIPv4';
  }
  const ttl = Number(form.ttl.trim());
  if (
    !form.proxied &&
    (!/^\d+$/.test(form.ttl.trim()) || (ttl !== 1 && (ttl < 30 || ttl > 86400)))
  ) {
    errors.ttl = 'invalidTTL';
  }
  return errors;
}

export function validateDNSCreateForm(form: DNSRecordForm & { name: string }) {
  return {
    ...validateDNSRecordForm(form),
    ...(!form.name.trim() || form.name.trim().length > 253 ? { name: 'invalidName' as const } : {}),
  };
}
