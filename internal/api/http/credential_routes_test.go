package transporthttp

import (
	"bytes"
	"encoding/json"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci"
	"gitee.com/leoninew/PomeloOrbit-go/internal/common/crypto"
)

const credentialRouteFernetKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestCredentialRoutes(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.JWT.SecretKey = credentialRouteFernetKey
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"
	plainData := "-----BEGIN OPENSSH PRIVATE KEY-----\nsecret\n-----END OPENSSH PRIVATE KEY-----\n"

	createRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, authedRequest(http.MethodPost, "/api/ci/credential?project_id="+projectId, bytes.NewBufferString(`{"name":"GitHub Token","type":"github_token","data":"`+plainData+`"}`), token))
	if createRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid raw newline JSON status 400, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}

	createBody, err := json.Marshal(map[string]string{"name": "GitHub Token", "type": "github_token", "data": plainData})
	if err != nil {
		t.Fatal(err)
	}
	createRecorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, authedRequest(http.MethodPost, "/api/ci/credential?project_id="+projectId, bytes.NewBuffer(createBody), token))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected credential create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	if strings.Contains(createRecorder.Body.String(), "data") || strings.Contains(createRecorder.Body.String(), "encrypted") {
		t.Fatalf("credential create leaked secret fields: %s", createRecorder.Body.String())
	}
	var created pomeloorbit.CredentialResp
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Name != "GitHub Token" || created.Type != "github_token" || created.CreatedAt == "" {
		t.Fatalf("unexpected created credential: %+v", &created)
	}
	stored, err := database.Queryx(`SELECT encrypted_data FROM credential WHERE id = ?`, created.Id)
	if err != nil {
		t.Fatal(err)
	}
	var storedData string
	if stored.Next() {
		if err := stored.Scan(&storedData); err != nil {
			_ = stored.Close()
			t.Fatal(err)
		}
	}
	_ = stored.Close()
	if storedData == "" || storedData == plainData {
		t.Fatalf("expected stored credential data to be encrypted")
	}
	decryptedStoredData, err := security.DecryptString(credentialRouteFernetKey, storedData)
	if err != nil {
		t.Fatal(err)
	}
	if decryptedStoredData != plainData {
		t.Fatalf("unexpected stored credential plaintext: %q", decryptedStoredData)
	}

	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, authedRequest(http.MethodGet, "/api/ci/credential?project_id="+projectId+"&page=1&per_page=20&search=github", nil, token))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected credential list status 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var credentials pomeloorbit.CredentialPaginatedResp
	if err := json.NewDecoder(listRecorder.Body).Decode(&credentials); err != nil {
		t.Fatal(err)
	}
	if credentials.Total != 1 || len(credentials.Items) != 1 || credentials.Items[0].Id != created.Id || credentials.PerPage != 20 {
		t.Fatalf("unexpected credential list: %+v", &credentials)
	}

	getRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(getRecorder, authedRequest(http.MethodGet, "/api/ci/credential/"+created.Id, nil, token))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected credential get status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}

	updatedPlainData := "updated-token-value"
	updateBody, err := json.Marshal(map[string]string{"name": "GitHub Token Updated", "data": updatedPlainData})
	if err != nil {
		t.Fatal(err)
	}
	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/ci/credential/"+created.Id, bytes.NewBuffer(updateBody), token))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected credential update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated pomeloorbit.CredentialResp
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "GitHub Token Updated" {
		t.Fatalf("unexpected updated credential: %+v", &updated)
	}

	exportRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(exportRecorder, authedRequest(http.MethodGet, "/api/ci/credential/"+created.Id+"/export", nil, token))
	if exportRecorder.Code != http.StatusOK {
		t.Fatalf("expected credential export status 200, got %d: %s", exportRecorder.Code, exportRecorder.Body.String())
	}
	var exported pomeloorbit.CredentialExportResp
	if err := json.NewDecoder(exportRecorder.Body).Decode(&exported); err != nil {
		t.Fatal(err)
	}
	if exported.Version != cisvc.CredentialExportVersion || exported.Name != "GitHub Token Updated" || exported.Type != "github_token" || exported.Data != updatedPlainData {
		t.Fatalf("unexpected exported credential: %+v", &exported)
	}

	deleteRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRecorder, authedRequest(http.MethodDelete, "/api/ci/credential/"+created.Id, nil, token))
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected credential delete status 204, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestCredentialImportRoute(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.JWT.SecretKey = credentialRouteFernetKey
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"

	body := bytes.NewBufferString(`{"name":"Imported","type":"git_ssh","data":"imported-secret"}`)
	createRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, authedRequest(http.MethodPost, "/api/ci/credential/import?project_id="+projectId, body, token))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected credential import status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created pomeloorbit.CredentialResp
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	exportRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(exportRecorder, authedRequest(http.MethodGet, "/api/ci/credential/"+created.Id+"/export", nil, token))
	if exportRecorder.Code != http.StatusOK {
		t.Fatalf("expected imported credential export status 200, got %d: %s", exportRecorder.Code, exportRecorder.Body.String())
	}
	var exported pomeloorbit.CredentialExportResp
	if err := json.NewDecoder(exportRecorder.Body).Decode(&exported); err != nil {
		t.Fatal(err)
	}
	if exported.Data != "imported-secret" || exported.Version != cisvc.CredentialExportVersion {
		t.Fatalf("unexpected imported credential export: %+v", &exported)
	}
}

func TestCredentialDuplicateName(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.JWT.SecretKey = credentialRouteFernetKey
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"
	first := createCredentialForTest(t, server, token, projectId, "Shared")

	duplicateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(duplicateRecorder, authedRequest(http.MethodPost, "/api/ci/credential?project_id="+projectId, bytes.NewBufferString(`{"name":"Shared","type":"github_token","data":"secret"}`), token))
	if duplicateRecorder.Code != http.StatusConflict {
		t.Fatalf("expected duplicate create status 409, got %d: %s", duplicateRecorder.Code, duplicateRecorder.Body.String())
	}
	second := createCredentialForTest(t, server, token, projectId, "Other")

	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/ci/credential/"+second.Id, bytes.NewBufferString(`{"name":"Shared"}`), token))
	if updateRecorder.Code != http.StatusConflict {
		t.Fatalf("expected duplicate update status 409, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	if first.Id == second.Id {
		t.Fatal("expected distinct credential ids")
	}
}

func TestCredentialDeleteReferenced(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.JWT.SecretKey = credentialRouteFernetKey
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"
	created := createCredentialForTest(t, server, token, projectId, "Referenced")
	if _, err := database.Exec(`UPDATE repository SET git_credential_id = ? WHERE id = '01KNNRBH52BQJYT9487B2H8N62'`, created.Id); err != nil {
		t.Fatal(err)
	}

	deleteRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRecorder, authedRequest(http.MethodDelete, "/api/ci/credential/"+created.Id, nil, token))
	if deleteRecorder.Code != http.StatusConflict {
		t.Fatalf("expected referenced credential delete status 409, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestCredentialValidation(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.JWT.SecretKey = credentialRouteFernetKey
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"

	missingProjectRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(missingProjectRecorder, authedRequest(http.MethodGet, "/api/ci/credential", nil, token))
	if missingProjectRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected missing project status 400, got %d: %s", missingProjectRecorder.Code, missingProjectRecorder.Body.String())
	}

	invalidJSONRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(invalidJSONRecorder, authedRequest(http.MethodPost, "/api/ci/credential?project_id="+projectId, bytes.NewBufferString(`{"name"`), token))
	if invalidJSONRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid json status 400, got %d: %s", invalidJSONRecorder.Code, invalidJSONRecorder.Body.String())
	}

	invalidFieldsRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(invalidFieldsRecorder, authedRequest(http.MethodPost, "/api/ci/credential?project_id="+projectId, bytes.NewBufferString(`{"name":"","type":"unknown","data":""}`), token))
	if invalidFieldsRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid fields status 400, got %d: %s", invalidFieldsRecorder.Code, invalidFieldsRecorder.Body.String())
	}
}

func TestCredentialRoutesRequireAuth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/ci/credential", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected credential list status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func createCredentialForTest(t *testing.T, server Server, token string, projectId string, name string) *pomeloorbit.CredentialResp {
	t.Helper()
	body := bytes.NewBufferString(`{"name":"` + name + `","type":"github_token","data":"secret"}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodPost, "/api/ci/credential?project_id="+projectId, body, token))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected credential create status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var created pomeloorbit.CredentialResp
	if err := json.NewDecoder(recorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	return &created
}
