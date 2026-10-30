package dnshandler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
	dnsv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/dns"
)

func TestDNSHandlersRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name    string
		handler gin.HandlerFunc
	}{
		{"zones", (Handler{}).ListZones}, {"records", (Handler{}).ListARecords}, {"update", (Handler{}).UpdateARecord}, {"create", (Handler{}).CreateARecord}, {"delete", (Handler{}).DeleteARecord},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/dns", tc.handler)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/dns", strings.NewReader(`{}`)))
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestDNSUpdateContractPreservesExplicitFalse(t *testing.T) {
	var input dnsv1.RecordUpdateReq
	if err := transport.UnmarshalProtoJSON([]byte(`{"content":"192.0.2.1","ttl":300,"proxied":false}`), &input); err != nil {
		t.Fatal(err)
	}
	if input.Proxied == nil || *input.Proxied {
		t.Fatalf("proxied=%v", input.Proxied)
	}
	var missing dnsv1.RecordUpdateReq
	if err := transport.UnmarshalProtoJSON([]byte(`{"content":"192.0.2.1","ttl":300}`), &missing); err != nil {
		t.Fatal(err)
	}
	if missing.Proxied != nil {
		t.Fatalf("missing proxied=%v", missing.Proxied)
	}
}
