package transport

import (
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

func WriteProtoJSON(c *gin.Context, status int, message proto.Message) {
	c.Render(status, protoJSONRenderer{message: message})
}
