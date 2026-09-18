export type LinuxSshCommandInput = {
  publicKey: string;
};

function bashSingleQuote(value: string): string {
  return `'${value.replace(/'/g, "'\"'\"'")}'`;
}

export function buildLinuxSshInitializationCommand(input: LinuxSshCommandInput): string {
  const publicKey = input.publicKey.trim();
  if (!publicKey) {
    return '';
  }
  const quotedKey = bashSingleQuote(publicKey);
  return `set -eu
umask 077
mkdir -p "$HOME/.ssh"
chmod 700 "$HOME/.ssh"
touch "$HOME/.ssh/authorized_keys"
chmod 600 "$HOME/.ssh/authorized_keys"
if ! grep -qxF -- ${quotedKey} "$HOME/.ssh/authorized_keys"; then
  printf '%s\\n' ${quotedKey} >> "$HOME/.ssh/authorized_keys"
fi`;
}
