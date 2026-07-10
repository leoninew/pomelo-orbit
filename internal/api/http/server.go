package transporthttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"

	"github.com/gin-gonic/gin"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/routes"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

type Server struct {
	appCfg config.Config
	cfg    config.ServerConfig
	logger *slog.Logger
	deps   routes.Dependencies
}

func New(cfg config.Config, logger *slog.Logger, deps routes.Dependencies) Server {
	return Server{appCfg: cfg, cfg: cfg.Server, logger: logger, deps: deps}
}

func (s Server) Handler() http.Handler {
	return routes.New(s.appCfg, s.logger, s.deps, s.registerFallbackRoutes).Handler()
}

func (s Server) registerFallbackRoutes(r *gin.Engine) {
	r.NoRoute(func(c *gin.Context) {
		if isAPIPath(c.Request.URL.Path, s.appCfg.Server.ApiPathPrefixes) {
			transportresponse.Error(c, http.StatusNotFound, "Not Found")
			return
		}
		if s.serveStatic(c, "static") {
			return
		}
		transportresponse.Error(c, http.StatusNotFound, "Not Found")
	})
}

func (s Server) serveStatic(c *gin.Context, staticDir string) bool {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return false
	}

	indexPath := filepath.Join(staticDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return false
	}

	requestPath := path.Clean("/" + c.Request.URL.Path)
	if requestPath != "/" {
		filePath := filepath.Join(staticDir, filepath.FromSlash(strings.TrimPrefix(requestPath, "/")))
		info, err := os.Stat(filePath)
		if err == nil && !info.IsDir() {
			if filepath.Clean(filePath) == filepath.Clean(indexPath) {
				return s.serveIndexHTML(c, indexPath)
			}
			http.ServeFile(c.Writer, c.Request, filePath)
			return true
		}
	}

	return s.serveIndexHTML(c, indexPath)
}

func (s Server) serveIndexHTML(c *gin.Context, indexPath string) bool {
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return false
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if c.Request.Method == http.MethodHead {
		return true
	}
	_, _ = c.Writer.Write(injectRuntimeConfig(content, s.appCfg.Server.PublicURL))
	return true
}

func injectRuntimeConfig(content []byte, publicURL string) []byte {
	configValue := map[string]string{}
	publicURL = strings.TrimRight(strings.TrimSpace(publicURL), "/")
	if publicURL != "" {
		configValue["publicUrl"] = publicURL
	}
	configJSON, err := json.Marshal(configValue)
	if err != nil {
		panic(fmt.Sprintf("marshal runtime config: %v", err))
	}
	script := []byte("<script>window.__CONFIG__ = " + string(configJSON) + ";</script>")
	placeholder := []byte("<!-- __RUNTIME_CONFIG__ -->")
	if bytes.Contains(content, placeholder) {
		return bytes.Replace(content, placeholder, script, 1)
	}
	headEnd := []byte("</head>")
	if bytes.Contains(content, headEnd) {
		return bytes.Replace(content, headEnd, append(script, headEnd...), 1)
	}
	return content
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
}

func isAPIPath(requestPath string, prefixes []string) bool {
	return hasAPIPathPrefix(requestPath, prefixes)
}

func hasAPIPathPrefix(requestPath string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/") {
			return true
		}
	}
	return false
}
