export function isValidRouteCode(code: string): boolean {
  return /^[a-z](?:[a-z0-9-]{0,30}[a-z0-9])?$/.test(code);
}

export function routeDomainSuffix(externalDomain: string, internalDomain: string): string {
  return externalDomain || internalDomain;
}

export function domainAfterCodeChange(
  previousCode: string,
  nextCode: string,
  domain: string,
  domainSuffix: string
): string {
  if (!domainSuffix) {
    return domain;
  }
  const previousGenerated = previousCode ? `${previousCode}.${domainSuffix}` : '';
  if (domain && domain !== previousGenerated) {
    return domain;
  }
  return isValidRouteCode(nextCode) ? `${nextCode}.${domainSuffix}` : '';
}
