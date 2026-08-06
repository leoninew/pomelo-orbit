package delivery

import (
	"context"
	"net/http"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpHTTPServerKey struct{}

// NewAuthenticatedStreamableHTTPHandler authenticates every HTTP request
// before it reaches a Streamable HTTP MCP session. serverForRequest must build
// a Core bound to the authenticated user represented by the request.
func NewAuthenticatedStreamableHTTPHandler(serverForRequest func(context.Context, string) (*mcp.Server, error)) http.Handler {
	streamable := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		server, _ := request.Context().Value(mcpHTTPServerKey{}).(*mcp.Server)
		return server
	}, nil)

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		server, err := serverForRequest(request.Context(), request.Header.Get("Authorization"))
		if err != nil {
			statusCode := http.StatusServiceUnavailable
			if apperror.IsKind(err, apperror.KindUnauthorized) {
				statusCode = http.StatusUnauthorized
			}
			http.Error(writer, http.StatusText(statusCode), statusCode)
			return
		}
		request = request.WithContext(context.WithValue(request.Context(), mcpHTTPServerKey{}, server))
		streamable.ServeHTTP(writer, request)
	})
}

// UnauthenticatedError avoids coupling the bootstrap composition root to the
// MCP adapter's HTTP error classification.
func UnauthenticatedError() error {
	return apperror.New(apperror.KindUnauthorized, "Not authenticated")
}
