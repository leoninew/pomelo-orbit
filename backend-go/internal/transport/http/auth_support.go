package transporthttp

import (
	"net/http"

	"backend/internal/config"
	"backend/internal/repository/model"
	"backend/internal/transport/http/handler/authz"
)

func (s Server) currentUser(w http.ResponseWriter, r *http.Request) (model.User, bool) {
	return authz.New(s.logger, s.userRepository, s.tokenService).CurrentUser(w, r)
}

func jwtSecret(cfg config.Config) string {
	return cfg.JWT.SecretKey
}
