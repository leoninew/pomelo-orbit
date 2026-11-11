package sshrunner

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/model"
	"golang.org/x/crypto/ssh"
)

func TestHostIdentityCallbacksHandleCertificateRenewalAndKeyChange(t *testing.T) {
	host, authority := testHostSigner(t), testHostSigner(t)
	first := testHostCertificate(t, host, authority, 1)
	renewed := testHostCertificate(t, host, authority, 2)
	other := testHostCertificate(t, testHostSigner(t), authority, 3)
	for _, policy := range []string{model.EnvironmentHostIdentityPolicyNegotiated, model.EnvironmentHostIdentityPolicyStable} {
		t.Run(policy, func(t *testing.T) {
			expected := ssh.FingerprintSHA256(first.PublicKey())
			if policy == model.EnvironmentHostIdentityPolicyStable {
				expected = ssh.FingerprintSHA256(host.PublicKey())
			}
			strict := strictHostKeyCallback(expected, policy)
			var observed string
			firstObservation := probeHostKeyCallback("", policy, &observed)
			for _, callback := range []ssh.HostKeyCallback{strict, firstObservation} {
				if err := callback("host:22", nil, first.PublicKey()); err != nil {
					t.Fatal(err)
				}
				for _, key := range []ssh.PublicKey{renewed.PublicKey(), host.PublicKey()} {
					err := callback("host:22", nil, key)
					if (err == nil) != (policy == model.EnvironmentHostIdentityPolicyStable) {
						t.Fatalf("same-key renewal or raw key: policy=%s err=%v", policy, err)
					}
				}
				if err := callback("host:22", nil, other.PublicKey()); err == nil {
					t.Fatal("changed public key accepted")
				}
			}
			if observed != expected {
				t.Fatalf("first observation changed: got=%q want=%q", observed, expected)
			}
			if hostKeyFingerprint(host.PublicKey(), policy) != ssh.FingerprintSHA256(host.PublicKey()) {
				t.Fatal("plain key fingerprint changed")
			}
		})
	}
}

func TestProbeAndRuntimeUseHostIdentityPolicyDuringHandshake(t *testing.T) {
	host, authority := testHostSigner(t), testHostSigner(t)
	first := testHostCertificate(t, host, authority, 1)
	renewed := testHostCertificate(t, host, authority, 2)
	for _, policy := range []string{model.EnvironmentHostIdentityPolicyNegotiated, model.EnvironmentHostIdentityPolicyStable} {
		t.Run(policy, func(t *testing.T) {
			target, _ := startSessionServerWithHostKey(t, renewed)
			target.Environment.SSH.HostIdentityPolicy = policy
			expected := ssh.FingerprintSHA256(renewed.PublicKey())
			previous := ssh.FingerprintSHA256(first.PublicKey())
			if policy == model.EnvironmentHostIdentityPolicyStable {
				expected = ssh.FingerprintSHA256(host.PublicKey())
				previous = expected
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			runtime := NewRuntime()
			prober := NewEnvironmentProber()
			for _, entrypoint := range []string{"probe", "runtime strict", "runtime observation"} {
				t.Run(entrypoint, func(t *testing.T) {
					target.Environment.SSH.HostKeyFingerprint = previous
					var err error
					var observed string
					if entrypoint == "probe" {
						observed, err = prober.Probe(ctx, target.Environment, *target.PrivateKey)
					} else {
						var observation *string
						if entrypoint == "runtime observation" {
							observation = &observed
						}
						client, dialErr := runtime.dialSSHWithHostKey(ctx, target, observation)
						err = dialErr
						if client != nil {
							_ = client.Close()
						}
					}
					if (err == nil) != (policy == model.EnvironmentHostIdentityPolicyStable) {
						t.Fatalf("renewed certificate handshake: policy=%s err=%v", policy, err)
					}
					if err == nil && entrypoint != "runtime strict" && observed != expected {
						t.Fatalf("observed=%q want=%q", observed, expected)
					}
				})
			}
			target.Environment.SSH.HostKeyFingerprint = ""
			observed, err := prober.Probe(ctx, target.Environment, *target.PrivateKey)
			if err != nil || observed != expected {
				t.Fatalf("probe first observation=%q err=%v", observed, err)
			}
			observed = ""
			client, err := runtime.dialSSHWithHostKey(ctx, target, &observed)
			if err != nil {
				t.Fatal(err)
			}
			_ = client.Close()
			if observed != expected {
				t.Fatalf("runtime first observation=%q want=%q", observed, expected)
			}
		})
	}
}

func testHostSigner(t *testing.T) ssh.Signer {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

func testHostCertificate(t *testing.T, host, authority ssh.Signer, serial uint64) ssh.Signer {
	t.Helper()
	certificate := &ssh.Certificate{Key: host.PublicKey(), CertType: ssh.HostCert, Serial: serial, ValidBefore: ssh.CertTimeInfinity}
	if err := certificate.SignCert(rand.Reader, authority); err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewCertSigner(certificate, host)
	if err != nil {
		t.Fatal(err)
	}
	return signer
}
