export type LinuxSshCommandInput = {
  username: string;
  workspaceRoot: string;
  publicKey: string;
  keyComment: string;
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
  const quotedComment = bashSingleQuote(input.keyComment);
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
key_file=$(mktemp "$HOME/.ssh/authorized_keys.XXXXXX")
trap 'rm -f -- "$key_file"' EXIT
ORBIT_SSH_KEY=${quotedKey} ORBIT_SSH_KEY_COMMENT=${quotedComment} awk '
BEGIN {
  key = ENVIRON["ORBIT_SSH_KEY"]
  comment = ENVIRON["ORBIT_SSH_KEY_COMMENT"]
}
{
  if ($0 == key) {
    if (have_previous && index(previous, "# Pomelo Orbit | ") != 1) print previous
    print comment
    print $0
    have_previous = 0
    found = 1
    next
  }
  if (have_previous) print previous
  previous = $0
  have_previous = 1
}
END {
  if (have_previous) print previous
  if (!found) {
    print comment
    print key
  }
}' "$HOME/.ssh/authorized_keys" > "$key_file"
cat "$key_file" > "$HOME/.ssh/authorized_keys"
rm -f -- "$key_file"
trap - EXIT
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
