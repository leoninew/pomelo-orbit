package routes

import (
	"github.com/gin-gonic/gin"
	dnshandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/dns"
)

func (r Router) registerDNS(engine *gin.Engine) {
	handler := dnshandler.New(r.deps.DNSService, r.deps.Authenticator)
	engine.GET("/api/dns/zone", handler.ListZones)
	engine.GET("/api/dns/zone/:zone_id/record", handler.ListARecords)
	engine.POST("/api/dns/zone/:zone_id/record", handler.CreateARecord)
	engine.DELETE("/api/dns/zone/:zone_id/record/:record_id", handler.DeleteARecord)
	engine.PUT("/api/dns/zone/:zone_id/record/:record_id", handler.UpdateARecord)
}
