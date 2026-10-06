function validIdentityToken(value: string): boolean {
  return value !== '' && !/[:\s\p{Cc}]/u.test(value);
}

export function validComponentUser(value: string): boolean {
  if (value === '') {
    return true;
  }
  const parts = value.split(':');
  return parts.length <= 2 && parts.every(validIdentityToken);
}

export function parseGroupAdd(value: string): string[] | null {
  if (value === '') {
    return [];
  }
  const groups = value.split(/[,\n]/).map((group) => group.trim());
  return groups.every(validIdentityToken) ? groups : null;
}
