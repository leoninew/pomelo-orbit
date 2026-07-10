package traefik

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
)

type MkcertGenerator struct{}

func (MkcertGenerator) Generate(ctx context.Context, domain string) (string, string, error) {
	if _, err := exec.LookPath("mkcert"); err != nil {
		return "", "", apperror.New(apperror.KindValidation, "mkcert 未安装或不可用,请参考 https://github.com/FiloSottile/mkcert#installation")
	}
	caRootOutput, err := exec.CommandContext(ctx, "mkcert", "-CAROOT").Output()
	if err != nil {
		return "", "", apperror.New(apperror.KindValidation, "mkcert CA 未安装到系统信任库。请先运行 mkcert -install 然后重启浏览器")
	}
	caRoot := strings.TrimSpace(string(caRootOutput))
	if caRoot == "" {
		return "", "", apperror.New(apperror.KindValidation, "mkcert CA 未安装到系统信任库。请先运行 mkcert -install 然后重启浏览器")
	}
	if _, err := os.Stat(filepath.Join(caRoot, "rootCA.pem")); err != nil {
		return "", "", apperror.New(apperror.KindValidation, "mkcert CA 未安装到系统信任库。请先运行 mkcert -install 然后重启浏览器")
	}
	tmp, err := os.MkdirTemp("", "pomelo-route-cert-*")
	if err != nil {
		return "", "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	certFile := filepath.Join(tmp, "cert.pem")
	keyFile := filepath.Join(tmp, "key.pem")
	cmd := exec.CommandContext(ctx, "mkcert", "-cert-file", certFile, "-key-file", keyFile, domain)
	if output, err := cmd.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", "", apperror.New(apperror.KindValidation, "mkcert 生成证书失败: "+message)
	}
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return "", "", err
	}
	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return "", "", err
	}
	return string(certPEM), string(keyPEM), nil
}
