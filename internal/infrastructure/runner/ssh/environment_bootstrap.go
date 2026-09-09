package sshrunner

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"golang.org/x/crypto/ssh"
)

const (
	defaultBootstrapTimeout      = 60 * time.Second
	maxBootstrapErrorOutputBytes = 4 * 1024
)

type bootstrapError struct {
	diagnostic string
}

func (e bootstrapError) Error() string {
	return e.diagnostic
}

func (e bootstrapError) BootstrapDiagnostic() string {
	return e.diagnostic
}

// EnvironmentBootstrapper initializes a reachable Linux OpenSSH target with
// the generated deployment public key. It never accepts an arbitrary command
// or allocates a PTY.
type EnvironmentBootstrapper struct {
	dialContext func(context.Context, string, string) (net.Conn, error)
	timeout     time.Duration
}

func NewEnvironmentBootstrapper() EnvironmentBootstrapper {
	dialer := net.Dialer{}
	return EnvironmentBootstrapper{dialContext: dialer.DialContext, timeout: defaultBootstrapTimeout}
}

func (b EnvironmentBootstrapper) Bootstrap(ctx context.Context, environment model.Environment, publicKey string, auth environmentport.BootstrapAuth) error {
	if !environment.IsSSH() || environment.SSH.Platform != model.EnvironmentPlatformLinux {
		return errors.New("SSH environment bootstrap received a non-Linux SSH environment")
	}
	if b.dialContext == nil {
		return errors.New("SSH bootstrap dialer is not configured")
	}
	command, err := linuxEnvironmentBootstrapCommand(*environment.SSH, publicKey)
	if err != nil {
		return err
	}
	authMethods, err := bootstrapAuthMethods(auth)
	if err != nil {
		return bootstrapError{diagnostic: "The supplied bootstrap SSH credential is invalid."}
	}
	if b.timeout <= 0 {
		b.timeout = defaultBootstrapTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	address := net.JoinHostPort(strings.TrimSpace(environment.SSH.Host), strconv.Itoa(environment.SSH.Port))
	connection, err := b.dialContext(ctx, "tcp", address)
	if err != nil {
		return bootstrapError{diagnostic: "Cannot connect to the configured SSH host."}
	}
	defer func() { _ = connection.Close() }()
	stopCancelClose := context.AfterFunc(ctx, func() { _ = connection.Close() })
	defer stopCancelClose()

	var observedFingerprint string
	clientConfig := &ssh.ClientConfig{
		User:            auth.Username,
		Auth:            authMethods,
		HostKeyCallback: probeHostKeyCallback(environment.SSH.HostKeyFingerprint, &observedFingerprint),
	}
	clientConnection, channels, requests, err := ssh.NewClientConn(connection, address, clientConfig)
	if err != nil {
		if strings.Contains(err.Error(), "SSH host key fingerprint mismatch") {
			return bootstrapError{diagnostic: "The SSH host key fingerprint does not match the configured host."}
		}
		return bootstrapError{diagnostic: "SSH authentication failed for the bootstrap user."}
	}
	client := ssh.NewClient(clientConnection, channels, requests)
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		return bootstrapError{diagnostic: "SSH connected, but a remote session could not be opened."}
	}
	defer func() { _ = session.Close() }()
	var stderr limitedBootstrapOutput
	session.Stdout = io.Discard
	session.Stderr = &stderr
	if err := session.Run(command); err != nil {
		return bootstrapError{diagnostic: bootstrapFailureDiagnostic(stderr.String())}
	}
	return nil
}

func bootstrapAuthMethods(auth environmentport.BootstrapAuth) ([]ssh.AuthMethod, error) {
	hasPassword := auth.Password != ""
	hasPrivateKey := strings.TrimSpace(auth.PrivateKey) != ""
	if strings.TrimSpace(auth.Username) == "" || hasPassword == hasPrivateKey {
		return nil, errors.New("exactly one bootstrap SSH credential is required")
	}
	if hasPassword {
		return []ssh.AuthMethod{ssh.Password(auth.Password)}, nil
	}
	privateKey := []byte(strings.TrimSpace(auth.PrivateKey))
	var signer ssh.Signer
	var err error
	if auth.PrivateKeyPassphrase == "" {
		signer, err = ssh.ParsePrivateKey(privateKey)
	} else {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(privateKey, []byte(auth.PrivateKeyPassphrase))
	}
	if err != nil {
		return nil, errors.New("parse bootstrap SSH private key")
	}
	return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
}

type limitedBootstrapOutput struct {
	buffer bytes.Buffer
}

func (b *limitedBootstrapOutput) Write(value []byte) (int, error) {
	originalLength := len(value)
	remaining := maxBootstrapErrorOutputBytes - b.buffer.Len()
	if remaining > 0 {
		if len(value) > remaining {
			value = value[:remaining]
		}
		_, _ = b.buffer.Write(value)
	}
	return originalLength, nil
}

func (b *limitedBootstrapOutput) String() string {
	return b.buffer.String()
}

func bootstrapFailureDiagnostic(stderr string) string {
	switch {
	case strings.Contains(stderr, "POMELO_ORBIT_BOOTSTRAP:DOCKER_PREREQUISITES_UNAVAILABLE"):
		return "Docker Engine and Docker Compose must already be running and usable by the configured deployment user."
	case strings.Contains(stderr, "POMELO_ORBIT_BOOTSTRAP:PASSWORDLESS_SUDO_REQUIRED"):
		return "The bootstrap SSH user must have passwordless sudo access."
	case strings.Contains(stderr, "POMELO_ORBIT_BOOTSTRAP:CONFIGURED_USER_MISSING"):
		return "The configured deployment SSH user does not exist on the target."
	case strings.Contains(stderr, "POMELO_ORBIT_BOOTSTRAP:OPENSSH_PUBLIC_KEY_AUTH_UNAVAILABLE"):
		return "OpenSSH public-key authentication is not enabled for the configured deployment user."
	case strings.Contains(stderr, "POMELO_ORBIT_BOOTSTRAP:WORKSPACE_UNAVAILABLE"):
		return "The deployment workspace cannot be created or written by the configured deployment user."
	default:
		return "Linux SSH initialization failed while validating OpenSSH or configuring the deployment key and workspace."
	}
}

func linuxEnvironmentBootstrapCommand(target model.EnvironmentSSHTarget, publicKey string) (string, error) {
	username := strings.TrimSpace(target.Username)
	workspaceRoot := strings.TrimSpace(target.WorkspaceRoot)
	publicKey = strings.TrimSpace(publicKey)
	if username == "" || workspaceRoot == "" || publicKey == "" {
		return "", errors.New("linux SSH bootstrap target is invalid")
	}
	quotedUsername := posixSingleQuote(username)
	quotedWorkspaceRoot := posixSingleQuote(workspaceRoot)
	quotedPublicKey := posixSingleQuote(publicKey)
	installKeyScript := `umask 077
if [ ! -d "$HOME/.ssh" ]; then
  mkdir "$HOME/.ssh"
fi
if [ "$(stat -c %a "$HOME/.ssh")" != 700 ]; then
  chmod 700 "$HOME/.ssh"
fi
if [ ! -f "$HOME/.ssh/authorized_keys" ]; then
  touch "$HOME/.ssh/authorized_keys"
fi
if [ "$(stat -c %a "$HOME/.ssh/authorized_keys")" != 600 ]; then
  chmod 600 "$HOME/.ssh/authorized_keys"
fi
if ! grep -qxF -- '` + quotedPublicKey + `' "$HOME/.ssh/authorized_keys"; then
  printf '%s\n' '` + quotedPublicKey + `' >> "$HOME/.ssh/authorized_keys"
fi`
	return `set -eu
fail() {
  printf '%s\n' "POMELO_ORBIT_BOOTSTRAP:$1" >&2
  exit 1
}
if ! command -v sudo >/dev/null 2>&1 || ! sudo -n -v >/dev/null 2>&1; then
  fail PASSWORDLESS_SUDO_REQUIRED
fi
if ! id -u '` + quotedUsername + `' >/dev/null 2>&1; then
  fail CONFIGURED_USER_MISSING
fi
if ! sudo -n -H -u '` + quotedUsername + `' sh -c 'command -v docker >/dev/null && docker version --format "{{.Server.Version}}" >/dev/null && docker compose version >/dev/null && docker info >/dev/null'; then
  fail DOCKER_PREREQUISITES_UNAVAILABLE
fi
if ! sudo -n sh -c 'command -v sshd >/dev/null 2>&1 && sshd -t'; then
  fail OPENSSH_PUBLIC_KEY_AUTH_UNAVAILABLE
fi
effectiveConfig=$(sudo -n sshd -T 2>/dev/null) || fail OPENSSH_PUBLIC_KEY_AUTH_UNAVAILABLE
if ! printf '%s\n' "$effectiveConfig" | awk '$1 == "pubkeyauthentication" && $2 == "yes" { found = 1 } END { exit !found }'; then
  fail OPENSSH_PUBLIC_KEY_AUTH_UNAVAILABLE
fi
if ! printf '%s\n' "$effectiveConfig" | awk '$1 == "authorizedkeysfile" && $0 ~ /\.ssh\/authorized_keys/ { found = 1 } END { exit !found }'; then
  fail OPENSSH_PUBLIC_KEY_AUTH_UNAVAILABLE
fi
sudo -n -H -u '` + quotedUsername + `' sh -c '` + posixSingleQuote(installKeyScript) + `'
targetHome=$(sudo -n -H -u '` + quotedUsername + `' sh -c 'printf %s "$HOME"')
workspaceRoot='` + quotedWorkspaceRoot + `'
case "$workspaceRoot" in
  '~') workspaceRoot=$targetHome ;;
  '~/'*) workspaceRoot=$targetHome/${workspaceRoot#~/} ;;
  /*) : ;;
  *) fail WORKSPACE_UNAVAILABLE ;;
esac
targetGroup=$(sudo -n id -gn '` + quotedUsername + `')
if [ ! -d "$workspaceRoot" ] || \
  [ "$(sudo -n stat -c %U "$workspaceRoot")" != '` + quotedUsername + `' ] || \
  [ "$(sudo -n stat -c %G "$workspaceRoot")" != "$targetGroup" ] || \
  [ "$(sudo -n stat -c %a "$workspaceRoot")" != 750 ]; then
  sudo -n install -d -m 0750 -o '` + quotedUsername + `' -g "$targetGroup" "$workspaceRoot" || fail WORKSPACE_UNAVAILABLE
fi
if ! sudo -n -H -u '` + quotedUsername + `' test -w "$workspaceRoot"; then
  fail WORKSPACE_UNAVAILABLE
fi`, nil
}

func posixSingleQuote(value string) string {
	return strings.ReplaceAll(value, "'", "'\"'\"'")
}
