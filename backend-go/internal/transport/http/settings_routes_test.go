package transporthttp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSettingsConfigRoutes(t *testing.T) {
	server, database := newTestServer(t)
	server.appCfg.Orbit.Root = t.TempDir()
	defer func() { _ = database.Close() }()
	token := testToken(t, server)

	getRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(getRecorder, authedRequest(http.MethodGet, "/api/settings/config", nil, token))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected settings get status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}
	var initial systemConfigResp
	if err := json.NewDecoder(getRecorder.Body).Decode(&initial); err != nil {
		t.Fatal(err)
	}
	if _, ok := findConfigItem(initial.Items, "logging__level"); !ok {
		t.Fatalf("expected logging__level in settings response: %+v", initial)
	}

	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/settings/config", bytes.NewBufferString(`{"key":"logging__level","value":"debug"}`), token))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected settings update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated systemConfigResp
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	loggingLevel, ok := findConfigItem(updated.Items, "logging__level")
	if !ok || loggingLevel.Value != "debug" || !loggingLevel.IsOverridden {
		t.Fatalf("unexpected updated logging level: %+v", loggingLevel)
	}

	resetRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(resetRecorder, authedRequest(http.MethodDelete, "/api/settings/config", bytes.NewBufferString(`{"keys":["logging__level"]}`), token))
	if resetRecorder.Code != http.StatusOK {
		t.Fatalf("expected settings reset status 200, got %d: %s", resetRecorder.Code, resetRecorder.Body.String())
	}
	var reset systemConfigResp
	if err := json.NewDecoder(resetRecorder.Body).Decode(&reset); err != nil {
		t.Fatal(err)
	}
	loggingLevel, ok = findConfigItem(reset.Items, "logging__level")
	if !ok || loggingLevel.Value != server.appCfg.Logging.Level || loggingLevel.IsOverridden {
		t.Fatalf("unexpected reset logging level: %+v", loggingLevel)
	}
}

func TestSettingsConfigRequiresAuth(t *testing.T) {
	server, database := newTestServer(t)
	server.appCfg.Orbit.Root = t.TempDir()
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/settings/config", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected settings get status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func findConfigItem(items []configItemResp, key string) (configItemResp, bool) {
	for _, item := range items {
		if item.Key == key {
			return item, true
		}
	}
	return configItemResp{}, false
}
