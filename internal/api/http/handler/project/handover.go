package projecthandler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
	handoverdto "github.com/leoninew/pomelo-orbit/internal/application/project_handover/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

const maxHandoverDocumentBytes int64 = 64 << 20

func (h Handler) ExportHandover(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if h.handover == nil {
		transport.WriteError(c, apperror.New(apperror.KindInternal, "Project handover service is not configured"))
		return
	}
	project, err := h.service.LoadForUser(c.Request.Context(), c.Param("project_id"), current.Id)
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	document, err := h.handover.ExportDocument(c.Request.Context(), current.Id, project.Id)
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	c.Header("Content-Type", "application/vnd.pomelo-orbit.project-handover+json")
	c.Header("Content-Disposition", "attachment; filename="+strconv.Quote(project.Code+".orbit-project-handover.json"))
	c.Data(http.StatusOK, "application/vnd.pomelo-orbit.project-handover+json", document)
}

func (h Handler) ImportHandover(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if h.handover == nil {
		transport.WriteError(c, apperror.New(apperror.KindInternal, "Project handover service is not configured"))
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxHandoverDocumentBytes)
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "handover file is required")
		return
	}
	defer func() { _ = file.Close() }()
	document, err := io.ReadAll(io.LimitReader(file, maxHandoverDocumentBytes+1))
	if err != nil || int64(len(document)) > maxHandoverDocumentBytes {
		transport.WriteStatusError(c, http.StatusBadRequest, "handover file is too large")
		return
	}
	project, err := h.handover.ImportDocument(
		c.Request.Context(),
		current.Id,
		handoverdto.ImportInput{
			Mode:            handoverdto.ImportMode(c.PostForm("mode")),
			Name:            c.PostForm("name"),
			Code:            c.PostForm("code"),
			TargetProjectId: c.Param("project_id"),
			DecryptionKey:   c.PostForm("decryption_key"),
		},
		document,
	)
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := projectResponse(project)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}
