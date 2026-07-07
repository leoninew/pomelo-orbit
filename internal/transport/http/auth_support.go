package transporthttp

import (
	"github.com/gin-gonic/gin"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/handler/authz"
)

func (s Server) currentUser(c *gin.Context) (model.User, bool) {
	return authz.New(s.logger, s.userRepository, s.tokenService).CurrentUser(c)
}

func jwtSecret(cfg config.Config) string {
	return cfg.JWT.SecretKey
}
