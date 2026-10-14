package routesvc

import (
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestPreviewRuleProtocolDescribesIngressRatherThanUpstream(t *testing.T) {
	for _, item := range []struct {
		protocol string
		https    bool
		target   string
	}{
		{protocol: "http", target: "https://upstream:8443"},
		{protocol: "https", https: true, target: "http://upstream:8080"},
	} {
		t.Run(item.protocol, func(t *testing.T) {
			rule := plannedRouteRule(model.Route{
				Protocol: "http", HTTPSEnabled: item.https, Domain: "api.example.test", PathPrefix: "/api", TargetUrl: item.target,
			})
			if rule.Protocol != item.protocol || rule.Match != "Host(`api.example.test`) && PathPrefix(`/api`)" || rule.Target != item.target {
				t.Fatalf("rule=%+v", rule)
			}
		})
	}
}
