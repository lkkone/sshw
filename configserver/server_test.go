package configserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestConfigServerPublishAndSyncFlow(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "test.db"), "test-master-key")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	app, err := NewServer(ServerConfig{
		Store:          store,
		AdminUsername:  "admin",
		AdminPassword:  "test-password",
		SessionSecret:  "test-session",
		DefaultProfile: "default",
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app.Handler())
	defer server.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	login := doJSON(t, client, http.MethodPost, server.URL+"/api/login", map[string]string{
		"username": "admin",
		"password": "test-password",
	})
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d: %s", login.StatusCode, readBody(login))
	}
	login.Body.Close()

	document := Document{Nodes: []*EditorNode{{
		Kind:  "host",
		Name:  "Production",
		Alias: "prod",
		Host:  "10.0.0.10",
		User:  "deploy",
		Port:  22,
	}}}
	draft := doJSON(t, client, http.MethodPut, server.URL+"/api/admin/draft", document)
	if draft.StatusCode != http.StatusOK {
		t.Fatalf("draft status = %d: %s", draft.StatusCode, readBody(draft))
	}
	draft.Body.Close()

	publish := doJSON(t, client, http.MethodPost, server.URL+"/api/admin/publish", map[string]string{"note": "Initial config"})
	if publish.StatusCode != http.StatusCreated {
		t.Fatalf("publish status = %d: %s", publish.StatusCode, readBody(publish))
	}
	publish.Body.Close()

	document.Nodes[0].Alias = "prod-v2"
	draft = doJSON(t, client, http.MethodPut, server.URL+"/api/admin/draft", document)
	if draft.StatusCode != http.StatusOK {
		t.Fatalf("second draft status = %d: %s", draft.StatusCode, readBody(draft))
	}
	draft.Body.Close()

	restore := doJSON(t, client, http.MethodPost, server.URL+"/api/admin/restore", map[string]int{"version": 1})
	if restore.StatusCode != http.StatusOK {
		t.Fatalf("restore status = %d: %s", restore.StatusCode, readBody(restore))
	}
	restore.Body.Close()
	state, err := store.State(t.Context(), "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Document.Nodes) != 1 || state.Document.Nodes[0].Alias != "prod" {
		t.Fatalf("restored draft = %#v", state.Document.Nodes)
	}

	tokenResponse := doJSON(t, client, http.MethodPost, server.URL+"/api/admin/tokens", map[string]string{"name": "Test laptop"})
	if tokenResponse.StatusCode != http.StatusCreated {
		t.Fatalf("token status = %d: %s", tokenResponse.StatusCode, readBody(tokenResponse))
	}
	var tokenPayload struct {
		Token  string     `json:"token"`
		Device TokenEntry `json:"device"`
	}
	if err := json.NewDecoder(tokenResponse.Body).Decode(&tokenPayload); err != nil {
		t.Fatal(err)
	}
	tokenResponse.Body.Close()

	request, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/sync/default", nil)
	request.Header.Set("Authorization", "Bearer "+tokenPayload.Token)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(response)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("sync status = %d: %s", response.StatusCode, body)
	}
	if !strings.Contains(body, "alias: prod") || response.Header.Get("X-SSHW-Version") != "1" {
		t.Fatalf("unexpected sync response: headers=%v body=%s", response.Header, body)
	}
	etag := response.Header.Get("ETag")

	request, _ = http.NewRequest(http.MethodGet, server.URL+"/api/v1/sync/default", nil)
	request.Header.Set("Authorization", "Bearer "+tokenPayload.Token)
	request.Header.Set("If-None-Match", etag)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotModified {
		t.Fatalf("conditional sync status = %d", response.StatusCode)
	}

	revoke := doJSON(t, client, http.MethodDelete, server.URL+"/api/admin/tokens/"+strconv.FormatInt(tokenPayload.Device.ID, 10), nil)
	if revoke.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke status = %d: %s", revoke.StatusCode, readBody(revoke))
	}
	revoke.Body.Close()
	request, _ = http.NewRequest(http.MethodGet, server.URL+"/api/v1/sync/default", nil)
	request.Header.Set("Authorization", "Bearer "+tokenPayload.Token)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked token status = %d", response.StatusCode)
	}
}

func TestConfigServerImportsExistingYAMLWithoutSavingIt(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "test.db"), "test-master-key")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	app, err := NewServer(ServerConfig{
		Store:          store,
		AdminUsername:  "admin",
		AdminPassword:  "test-password",
		SessionSecret:  "test-session",
		DefaultProfile: "default",
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app.Handler())
	defer server.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	login := doJSON(t, client, http.MethodPost, server.URL+"/api/login", map[string]string{
		"username": "admin",
		"password": "test-password",
	})
	login.Body.Close()

	yamlData := `
- name: Production
  children:
    - name: API
      alias: prod-api
      host: 10.0.0.10
      user: deploy
      port: 22
      jump:
        - host: bastion.example.com
          user: jump
`
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/admin/import", strings.NewReader(yamlData))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/yaml")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("import status = %d: %s", response.StatusCode, readBody(response))
	}
	var imported struct {
		Document Document `json:"document"`
		Stats    struct {
			Hosts  int `json:"hosts"`
			Groups int `json:"groups"`
		} `json:"stats"`
	}
	if err := json.NewDecoder(response.Body).Decode(&imported); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if imported.Stats.Hosts != 1 || imported.Stats.Groups != 1 {
		t.Fatalf("imported stats = %#v", imported.Stats)
	}
	if len(imported.Document.Nodes) != 1 ||
		len(imported.Document.Nodes[0].Children) != 1 ||
		imported.Document.Nodes[0].Children[0].Alias != "prod-api" ||
		len(imported.Document.Nodes[0].Children[0].Jump) != 1 ||
		imported.Document.Nodes[0].Children[0].Jump[0].Name != "" {
		t.Fatalf("imported document = %#v", imported.Document)
	}

	state, err := store.State(t.Context(), "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Document.Nodes) != 0 {
		t.Fatalf("import unexpectedly saved draft: %#v", state.Document.Nodes)
	}
}

func TestConfigServerRejectsInvalidImportedYAML(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "test.db"), "test-master-key")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	app, err := NewServer(ServerConfig{
		Store:          store,
		AdminUsername:  "admin",
		AdminPassword:  "test-password",
		SessionSecret:  "test-session",
		DefaultProfile: "default",
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app.Handler())
	defer server.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	login := doJSON(t, client, http.MethodPost, server.URL+"/api/login", map[string]string{
		"username": "admin",
		"password": "test-password",
	})
	login.Body.Close()

	request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/admin/import", strings.NewReader("- name: Missing host\n"))
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(response)
	if response.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, "host is required") {
		t.Fatalf("invalid import status = %d body = %s", response.StatusCode, body)
	}
}

func doJSON(t *testing.T, client *http.Client, method, endpoint string, value interface{}) *http.Response {
	t.Helper()
	var body io.Reader
	if value != nil {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(data)
	}
	request, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func readBody(response *http.Response) string {
	data, _ := io.ReadAll(response.Body)
	response.Body.Close()
	return string(data)
}
