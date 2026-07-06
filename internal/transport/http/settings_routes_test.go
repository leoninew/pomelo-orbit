package transporthttp

import (
	apiv1 "backend/internal/gen/orbit/api/v1"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	settingssvc "backend/internal/service/settings"
)

func TestSettingsConfigRoutes(t *testing.T) {
	server, database := newTestServer(t)
	envFilePath := filepath.Join(t.TempDir(), ".env")
	server.appCfg.Orbit.Root = t.TempDir()
	server.appCfg.EnvFilePath = envFilePath
	server.settingsService = settingssvc.New(server.appCfg)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)

	getRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(getRecorder, authedRequest(http.MethodGet, "/api/settings/config", nil, token))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected settings get status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}
	var initial apiv1.SystemConfigResp
	if err := json.NewDecoder(getRecorder.Body).Decode(&initial); err != nil {
		t.Fatal(err)
	}
	if _, ok := findConfigItem(initial.Items, "logging__level"); !ok {
		t.Fatalf("expected logging__level in settings response: %+v", &initial)
	}

	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/settings/config", bytes.NewBufferString(`{"key":"logging__level","value":"debug"}`), token))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected settings update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated apiv1.SystemConfigResp
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	loggingLevel, ok := findConfigItem(updated.Items, "logging__level")
	if !ok || loggingLevel.Value.AsInterface() != "debug" || !loggingLevel.IsOverridden {
		t.Fatalf("unexpected updated logging level: %+v", loggingLevel)
	}
	content, err := os.ReadFile(envFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte("POMELO_ORBIT_LOGGING__LEVEL=debug")) {
		t.Fatalf("expected settings update in env file %s: %s", envFilePath, string(content))
	}

	resetRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(resetRecorder, authedRequest(http.MethodDelete, "/api/settings/config", bytes.NewBufferString(`{"keys":["logging__level"]}`), token))
	if resetRecorder.Code != http.StatusOK {
		t.Fatalf("expected settings reset status 200, got %d: %s", resetRecorder.Code, resetRecorder.Body.String())
	}
	var reset apiv1.SystemConfigResp
	if err := json.NewDecoder(resetRecorder.Body).Decode(&reset); err != nil {
		t.Fatal(err)
	}
	loggingLevel, ok = findConfigItem(reset.Items, "logging__level")
	if !ok || loggingLevel.Value.AsInterface() != server.appCfg.Logging.Level || loggingLevel.IsOverridden {
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

func findConfigItem(items []*apiv1.ConfigItemResp, key string) (*apiv1.ConfigItemResp, bool) {
	for _, item := range items {
		if item != nil && item.Key == key {
			return item, true
		}
	}
	return nil, false
}
