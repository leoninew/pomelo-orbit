package routesvc

import (
	"strings"
	"testing"
)

func TestRouteValidationSeparatesIdentityFromCustomTargetUrl(t *testing.T) {
	if !validRouteCode("a"+strings.Repeat("b", 31)) || !validRouteCode("api1") || validRouteCode("1-api") || validRouteCode("123") || validRouteCode(strings.Repeat("a", 33)) || validRouteCode("bad_name") || validRouteCode("bad.name") || validRouteCode("bad-") {
		t.Fatal("route code DNS label validation is inconsistent")
	}
	validTargets := []string{
		"http://host",
		"https://host",
		"http://host:8080",
		"https://api.example.test:443",
	}
	for _, target := range validTargets {
		if !validRouteIdentity("api-route", "api.example.test", "/") || !routeTargetUrlPattern.MatchString(target) {
			t.Errorf("expected target URL to be valid: %s", target)
		}
	}

	invalidTargets := []string{
		"host:8080",
		"http://",
		"http://host/path",
		"ftp://host",
		"http://host:",
	}
	for _, target := range invalidTargets {
		if routeTargetUrlPattern.MatchString(target) {
			t.Errorf("expected target URL to be invalid: %s", target)
		}
	}
}
