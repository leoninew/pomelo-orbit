package bootstrap

import "testing"

func TestSSHUsernameHintStripsWindowsDomainPrefix(t *testing.T) {
	if got := sshUsernameHint(`DESKTOP-ABC\orbit`); got != "orbit" {
		t.Fatalf("windows username = %q", got)
	}
	if got := sshUsernameHint("orbit"); got != "orbit" {
		t.Fatalf("plain username = %q", got)
	}
}
