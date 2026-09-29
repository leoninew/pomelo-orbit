export function isValidRouteCode(code: string): boolean {
  return /^[a-z](?:[a-z0-9-]{0,30}[a-z0-9])?$/.test(code);
}

export function domainAfterCodeChange(
  previousCode: string,
  nextCode: string,
  domain: string,
  externalDomain: string
): string {
  if (!externalDomain) {
    return domain;
  }
  const previousGenerated = previousCode ? `${previousCode}.${externalDomain}` : '';
  if (domain && domain !== previousGenerated) {
    return domain;
  }
  return isValidRouteCode(nextCode) ? `${nextCode}.${externalDomain}` : '';
}
