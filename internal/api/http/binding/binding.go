package binding

import (
	"encoding/json"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
)

func DecodeJSON(c *gin.Context, value any) error {
	return DecodeJSONReader(c.Request.Body, value)
}

func DecodeJSONReader(r io.Reader, value any) error {
	if message, ok := value.(proto.Message); ok {
		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		return codec.UnmarshalProtoJSON(data, message)
	}
	return json.NewDecoder(r).Decode(value)
}

func QueryInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func QueryProjectId(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
