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
	"sort"
	"strings"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"

	"github.com/gin-gonic/gin"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/routes"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

type Server struct {
	appCfg config.Config
	cfg    config.ServerConfig
	logger *slog.Logger
	deps   routes.Dependencies
	mcp    http.Handler
}

func New(cfg config.Config, logger *slog.Logger, deps routes.Dependencies, mcpHandlers ...http.Handler) Server {
	var mcpHandler http.Handler
	if len(mcpHandlers) > 0 {
		mcpHandler = mcpHandlers[0]
	}
	return Server{appCfg: cfg, cfg: cfg.Server, logger: logger, deps: deps, mcp: mcpHandler}
}

func (s Server) Handler() http.Handler {
	webHandler := routes.New(s.appCfg, s.logger, s.deps, s.registerFallbackRoutes).Handler()
	if s.mcp == nil {
		return webHandler
	}
	mux := http.NewServeMux()
	mux.Handle("/mcp", s.mcp)
	mux.Handle("/", webHandler)
	return mux
}

func (s Server) registerFallbackRoutes(r *gin.Engine) {
	r.NoRoute(func(c *gin.Context) {
		if isApiPath(c.Request.URL.Path, s.appCfg.Server.ApiPathPrefixes) {
			transportresponse.WriteError(c, apperror.New(apperror.KindNotFound, ""))
			return
		}
		if s.serveStatic(c, "static") {
			return
		}
		transportresponse.WriteError(c, apperror.New(apperror.KindNotFound, ""))
	})
	r.NoMethod(func(c *gin.Context) {
		if methods := allowedHTTPMethods(r.Routes(), c.Request.URL.Path); len(methods) > 0 {
			c.Header("Allow", strings.Join(methods, ", "))
		}
		transportresponse.WriteError(c, apperror.New(apperror.KindMethodNotAllowed, ""))
	})
}

func allowedHTTPMethods(routes []gin.RouteInfo, requestPath string) []string {
	methods := make(map[string]struct{})
	for _, route := range routes {
		if routePathMatches(route.Path, requestPath) {
			methods[route.Method] = struct{}{}
		}
	}
	result := make([]string, 0, len(methods))
	for method := range methods {
		result = append(result, method)
	}
	sort.Strings(result)
	return result
}

func routePathMatches(routePath string, requestPath string) bool {
	routeSegments := pathSegments(routePath)
	requestSegments := pathSegments(requestPath)
	for index, routeSegment := range routeSegments {
		if strings.HasPrefix(routeSegment, "*") {
			return true
		}
		if index >= len(requestSegments) {
			return false
		}
		if strings.HasPrefix(routeSegment, ":") {
			if requestSegments[index] == "" {
				return false
			}
			continue
		}
		if routeSegment != requestSegments[index] {
			return false
		}
	}
	return len(routeSegments) == len(requestSegments)
}

func pathSegments(value string) []string {
	value = strings.Trim(value, "/")
	if value == "" {
		return nil
	}
	return strings.Split(value, "/")
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
	_, _ = c.Writer.Write(injectRuntimeConfig(content, s.appCfg.Server.PublicUrl))
	return true
}

func injectRuntimeConfig(content []byte, publicUrl string) []byte {
	configValue := map[string]string{}
	publicUrl = strings.TrimRight(strings.TrimSpace(publicUrl), "/")
	if publicUrl != "" {
		configValue["publicUrl"] = publicUrl
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

func isApiPath(requestPath string, prefixes []string) bool {
	return hasApiPathPrefix(requestPath, prefixes)
}

func hasApiPathPrefix(requestPath string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/") {
			return true
		}
	}
	return false
}
