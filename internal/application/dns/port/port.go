package dnsport

import (
	"context"

	dnsdto "github.com/leoninew/pomelo-orbit/internal/application/dns/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type ProjectReader interface {
	Project(context.Context, string) (model.Project, error)
	IsProjectMember(context.Context, string, string) (bool, error)
}

type GatewayReader interface {
	GatewayConfigByProject(context.Context, string) (model.GatewayConfig, error)
}

type Provider interface {
	ListZones(context.Context, string) ([]dnsdto.Zone, error)
	ListARecords(context.Context, string, string, string, int, int) (repository.Page[dnsdto.Record], error)
	CreateARecord(context.Context, string, string, dnsdto.CreateRecordInput) (dnsdto.Record, error)
	DeleteARecord(context.Context, string, string, string) error
	Record(context.Context, string, string, string) (dnsdto.Record, error)
	UpdateARecord(context.Context, string, string, string, string, dnsdto.UpdateRecordInput) (dnsdto.Record, error)
}
