package transport

import (
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

func DecodeJSON(c *gin.Context, message proto.Message) error {
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	return UnmarshalProtoJSON(data, message)
}

func QueryInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
