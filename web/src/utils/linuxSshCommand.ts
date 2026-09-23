export type LinuxSshCommandInput = {
  username: string;
  workspaceRoot: string;
  publicKey: string;
};

function bashSingleQuote(value: string): string {
  return `'${value.replace(/'/g, "'\"'\"'")}'`;
}

export function buildLinuxSshInitializationCommand(input: LinuxSshCommandInput): string {
  const publicKey = input.publicKey.trim();
  const username = input.username.trim();
  const workspaceRoot = input.workspaceRoot.trim();
  if (!publicKey || !username || !workspaceRoot) {
    return '';
  }
  const quotedKey = bashSingleQuote(publicKey);
  const quotedUsername = bashSingleQuote(username);
  const quotedWorkspace =
    workspaceRoot === '~'
      ? '"$HOME"'
      : workspaceRoot.startsWith('~/')
        ? `"$HOME"/${bashSingleQuote(workspaceRoot.slice(2))}`
        : bashSingleQuote(workspaceRoot);
  return `set -eu
if [ "$(id -un)" != ${quotedUsername} ]; then
  printf 'Run this command as the configured SSH user: %s\\n' ${quotedUsername} >&2
  exit 1
fi
umask 077
mkdir -p "$HOME/.ssh"
chmod 700 "$HOME/.ssh"
touch "$HOME/.ssh/authorized_keys"
chmod 600 "$HOME/.ssh/authorized_keys"
if ! grep -qxF -- ${quotedKey} "$HOME/.ssh/authorized_keys"; then
  printf '%s\\n' ${quotedKey} >> "$HOME/.ssh/authorized_keys"
fi
workspace_root=${quotedWorkspace}
run_as_admin() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
  else
    sudo "$@"
  fi
}
if [ ! -d "$workspace_root" ]; then
  run_as_admin mkdir -p -- "$workspace_root/pipeline" "$workspace_root/deployment"
  run_as_admin chown -- "$(id -u):$(id -g)" "$workspace_root" "$workspace_root/pipeline" "$workspace_root/deployment"
else
  run_as_admin mkdir -p -- "$workspace_root/pipeline" "$workspace_root/deployment"
  run_as_admin chown -- "$(id -u):$(id -g)" "$workspace_root/pipeline" "$workspace_root/deployment"
fi`;
}
