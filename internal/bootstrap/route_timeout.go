package bootstrap

import (
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	"github.com/leoninew/pomelo-orbit/internal/config"
)

func routeSyncTimeouts(cfg config.RouteConfig) routeport.SyncTimeouts {
	return routeport.SyncTimeouts{
		GatewayLock: cfg.GatewayLockTimeout, StateLoad: cfg.StateLoadTimeout, FilePublication: cfg.FilePublicationTimeout,
		ApiRequest: cfg.ApiRequestTimeout, Reload: cfg.ReloadTimeout,
		ConfigurationMatch: cfg.ConfigurationMatchTimeout, Recovery: cfg.RecoveryTimeout,
	}
}
