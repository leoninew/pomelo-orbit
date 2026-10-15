package environmenthandler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/terminal"
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
	environmentv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/environment"
)

const terminalProtocol = "orbit-terminal-v1"

func (h Handler) IssueTerminalTicket(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	ticket, err := h.terminal.IssueTicket(c.Request.Context(), current.Id, strings.TrimSpace(c.Query("project_id")))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	transport.WriteProtoJSON(c, http.StatusOK, &environmentv1.EnvironmentTerminalTicketResp{Ticket: ticket})
}

func (h Handler) ConnectTerminal(c *gin.Context) {
	projectId := strings.TrimSpace(c.Query("project_id"))
	token := terminalTicketProtocol(c.Request.Header.Values("Sec-WebSocket-Protocol"))
	if token == "" || projectId == "" {
		transport.WriteStatusError(c, http.StatusUnauthorized, "Invalid terminal ticket")
		return
	}
	grant, release, err := h.terminal.RedeemTicket(c.Request.Context(), projectId, token)
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	defer release()
	terminal.Serve(c, grant, h.terminal, h.runner, h.logger, h.origins)
}

func terminalTicketProtocol(values []string) string {
	foundProtocol := false
	ticket := ""
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == terminalProtocol {
				foundProtocol = true
			} else if strings.HasPrefix(part, "ticket.") && ticket == "" {
				ticket = strings.TrimPrefix(part, "ticket.")
			}
		}
	}
	if foundProtocol {
		return ticket
	}
	return ""
}
