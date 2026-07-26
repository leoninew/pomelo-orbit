package bootstrap

import (
	"database/sql"

	applicationrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/application"
	authrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/auth"
	credentialrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/credential"
	deploymentrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/deployment"
	gatewayrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/gateway"
	pipelinerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/pipeline"
	pipelinerunrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/pipeline_run"
	projectrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/project"
	vcsrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/repository"
	rolerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/role"
	routerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/route"
	servicerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/service"
	userrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/user"
)

// domainStores holds per-domain sqlc repositories for HTTP and worker wiring.
// Keep construction here so bootstrap paths cannot drift domain by domain.
type domainStores struct {
	user        userrepo.Repository
	auth        authrepo.Repository
	role        rolerepo.Repository
	project     projectrepo.Repository
	credential  credentialrepo.Repository
	repository  vcsrepo.Repository
	pipeline    pipelinerepo.Repository
	pipelineRun pipelinerunrepo.Repository
	application applicationrepo.Repository
	service     servicerepo.Repository
	deployment  deploymentrepo.Repository
	gateway     gatewayrepo.Repository
	route       routerepo.Repository
}

func newDomainStores(database *sql.DB) domainStores {
	return domainStores{
		user:        userrepo.NewRepository(database),
		auth:        authrepo.NewRepository(database),
		role:        rolerepo.NewRepository(database),
		project:     projectrepo.NewRepository(database),
		credential:  credentialrepo.NewRepository(database),
		repository:  vcsrepo.NewRepository(database),
		pipeline:    pipelinerepo.NewRepository(database),
		pipelineRun: pipelinerunrepo.NewRepository(database),
		application: applicationrepo.NewRepository(database),
		service:     servicerepo.NewRepository(database),
		deployment:  deploymentrepo.NewRepository(database),
		gateway:     gatewayrepo.NewRepository(database),
		route:       routerepo.NewRepository(database),
	}
}
