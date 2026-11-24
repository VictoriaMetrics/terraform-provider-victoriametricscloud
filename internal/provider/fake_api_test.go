package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	vmcloudapi "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
)

// fakeAPI is an in-memory VictoriaMetrics Cloud API for tests. It mirrors the
// behaviour of the real API that the provider depends on:
//   - a new deployment is PROVISIONING for a few reads and refuses updates until it
//     is RUNNING;
//   - the create request carries no flags, only updates do;
//   - an update that changes nothing the API compares is refused with 400
//     "there are no changes";
//   - rule files of a deployment that is not RUNNING are refused with 423;
//   - missing objects are answered with 404.
//
// With reportFlags the API returns the flags of a deployment and compares them on
// update, as it is meant to. Without it the API neither returns nor compares flags,
// as the API does today.
type fakeAPI struct {
	server      *httptest.Server
	reportFlags bool
	// provisioningReads is how many reads a new deployment stays PROVISIONING for.
	provisioningReads int

	mu          sync.Mutex
	nextID      int
	deployments map[string]*fakeDeployment
	tokens      map[string]map[string]vmcloudapi.AccessToken
	ruleFiles   map[string]map[string]string
}

type fakeDeployment struct {
	info  vmcloudapi.DeploymentInfo
	flags vmcloudapi.DeploymentFlags
	// provisioningReads is the number of reads left before the deployment is RUNNING.
	provisioningReads int
}

// defaultProvisioningReads is how many reads a new deployment stays PROVISIONING for.
const defaultProvisioningReads = 2

func newFakeAPI(t *testing.T, reportFlags bool) *fakeAPI {
	t.Helper()
	api := &fakeAPI{
		reportFlags:       reportFlags,
		provisioningReads: defaultProvisioningReads,
		deployments:       map[string]*fakeDeployment{},
		tokens:            map[string]map[string]vmcloudapi.AccessToken{},
		ruleFiles:         map[string]map[string]string{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/deployments", api.listDeployments)
	mux.HandleFunc("POST /api/v1/deployments", api.createDeployment)
	mux.HandleFunc("GET /api/v1/deployments/{id}", api.getDeployment)
	mux.HandleFunc("PUT /api/v1/deployments/{id}", api.updateDeployment)
	mux.HandleFunc("DELETE /api/v1/deployments/{id}", api.deleteDeployment)
	mux.HandleFunc("POST /api/v1/deployments/{id}/access_tokens", api.createToken)
	mux.HandleFunc("GET /api/v1/deployments/{id}/access_tokens/{token}", api.getToken)
	mux.HandleFunc("DELETE /api/v1/deployments/{id}/access_tokens/{token}", api.deleteToken)
	mux.HandleFunc("GET /api/v1/deployments/{id}/rule-sets/files/{name}", api.getRuleFile)
	mux.HandleFunc("POST /api/v1/deployments/{id}/rule-sets/files/{name}", api.putRuleFile)
	mux.HandleFunc("DELETE /api/v1/deployments/{id}/rule-sets/files/{name}", api.deleteRuleFile)
	api.server = httptest.NewServer(mux)
	t.Cleanup(api.server.Close)
	return api
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, format string, args ...any) {
	writeJSON(w, status, map[string]string{"error": fmt.Sprintf(format, args...)})
}

// advance moves a provisioning deployment one read closer to RUNNING.
func (d *fakeDeployment) advance() {
	if d.info.Status != vmcloudapi.DeploymentStatusProvisioning {
		return
	}
	d.provisioningReads--
	if d.provisioningReads <= 0 {
		d.info.Status = vmcloudapi.DeploymentStatusRunning
	}
}

// setStatus sets the status of deployment id.
func (a *fakeAPI) setStatus(id string, status vmcloudapi.DeploymentStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.deployments[id].info.Status = status
}

// deploymentFlags returns the flags the API stores for deployment id.
func (a *fakeAPI) deploymentFlags(id string) (vmcloudapi.DeploymentFlags, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	d, ok := a.deployments[id]
	if !ok {
		return vmcloudapi.DeploymentFlags{}, false
	}
	return d.flags, true
}

// removeDeployment deletes a deployment as if it was deleted outside Terraform.
func (a *fakeAPI) removeDeployment(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.deployments, id)
}

// removeToken deletes an access token as if it was deleted outside Terraform.
func (a *fakeAPI) removeToken(deploymentID, tokenID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.tokens[deploymentID], tokenID)
}

// removeRuleFile deletes a rule file as if it was deleted outside Terraform.
func (a *fakeAPI) removeRuleFile(deploymentID, name string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.ruleFiles[deploymentID], name)
}

// details returns what the API reports for d.
func (a *fakeAPI) details(d *fakeDeployment) vmcloudapi.DeploymentInfo {
	info := d.info
	if a.reportFlags {
		info.VMSingleSettings = d.flags.SingleFlags
		info.VMSelectSettings = d.flags.SelectFlags
		info.VMStorageSettings = d.flags.StorageFlags
		info.VMInsertSettings = d.flags.InsertFlags
	}
	return info
}

func storageGB(size uint64, unit vmcloudapi.StorageUnit) uint64 {
	if unit == "TB" {
		return size * 1024
	}
	return size
}

func (a *fakeAPI) listDeployments(w http.ResponseWriter, _ *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	list := vmcloudapi.DeploymentSummaryList{}
	for _, d := range a.deployments {
		list = append(list, vmcloudapi.DeploymentSummary{ID: d.info.ID, Name: d.info.Name, Type: d.info.Type, Status: d.info.Status})
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *fakeAPI) createDeployment(w http.ResponseWriter, r *http.Request) {
	var req vmcloudapi.DeploymentCreationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: %s", err)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nextID++
	id := fmt.Sprintf("00000000-0000-0000-0000-%012d", a.nextID)
	d := &fakeDeployment{
		info: vmcloudapi.DeploymentInfo{
			ID:                 id,
			Name:               req.Name,
			Type:               req.Type,
			Tier:               req.Tier,
			Version:            "v1.0.0",
			CloudProvider:      req.Provider,
			Region:             req.Region,
			CreatedAt:          time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			Status:             vmcloudapi.DeploymentStatusProvisioning,
			RetentionValue:     req.Retention,
			RetentionUnit:      req.RetentionUnit,
			DeduplicationValue: req.Deduplication,
			DeduplicationUnit:  req.DeduplicationUnit,
			StorageSizeGb:      storageGB(req.StorageSize, req.StorageSizeUnit),
			MaintenanceWindow:  req.MaintenanceWindow,
			AccessEndpoint:     "https://" + id + ".example.com",
		},
		provisioningReads: a.provisioningReads,
	}
	a.deployments[id] = d
	writeJSON(w, http.StatusOK, a.details(d))
}

func (a *fakeAPI) getDeployment(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	d, ok := a.deployments[r.PathValue("id")]
	if !ok {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}
	d.advance()
	writeJSON(w, http.StatusOK, a.details(d))
}

func (a *fakeAPI) updateDeployment(w http.ResponseWriter, r *http.Request) {
	var req vmcloudapi.DeploymentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: %s", err)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	d, ok := a.deployments[r.PathValue("id")]
	if !ok {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}
	if d.info.Status != vmcloudapi.DeploymentStatusRunning {
		writeError(w, http.StatusBadRequest, "deployment in readonly status please wait for update process will be finished and retry your request")
		return
	}
	if d.info.Type.SupportsDeduplication() && req.DeduplicationUnit == nil {
		writeError(w, http.StatusBadRequest, "field 'Deduplication unit': cannot be blank.")
		return
	}

	updated := d.info
	updated.Name = req.Name
	updated.Tier = req.Tier
	updated.StorageSizeGb = storageGB(req.StorageSize, req.StorageSizeUnit)
	updated.RetentionValue = req.Retention
	updated.RetentionUnit = req.RetentionUnit
	updated.MaintenanceWindow = req.MaintenanceWindow
	if req.Deduplication != nil {
		updated.DeduplicationValue = req.Deduplication
		updated.DeduplicationUnit = req.DeduplicationUnit
	}
	unchanged := reflect.DeepEqual(updated, d.info)
	if a.reportFlags {
		unchanged = unchanged && reflect.DeepEqual(normalizeFlags(req.Flags), normalizeFlags(d.flags))
	}
	if unchanged {
		writeError(w, http.StatusBadRequest, "there are no changes in deployment. Please change any field to update the deployment")
		return
	}
	d.info = updated
	d.flags = normalizeFlags(req.Flags)
	writeJSON(w, http.StatusOK, a.details(d))
}

// normalizeFlags stores unset flags as empty lists, as the API reports them.
func normalizeFlags(f vmcloudapi.DeploymentFlags) vmcloudapi.DeploymentFlags {
	norm := func(l vmcloudapi.FlagList) vmcloudapi.FlagList {
		if l == nil {
			return vmcloudapi.FlagList{}
		}
		return l
	}
	return vmcloudapi.DeploymentFlags{
		SingleFlags:  norm(f.SingleFlags),
		SelectFlags:  norm(f.SelectFlags),
		StorageFlags: norm(f.StorageFlags),
		InsertFlags:  norm(f.InsertFlags),
	}
}

func (a *fakeAPI) deleteDeployment(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	id := r.PathValue("id")
	if _, ok := a.deployments[id]; !ok {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}
	delete(a.deployments, id)
	writeJSON(w, http.StatusOK, map[string]string{})
}

func (a *fakeAPI) createToken(w http.ResponseWriter, r *http.Request) {
	var req vmcloudapi.AccessTokenCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: %s", err)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	deploymentID := r.PathValue("id")
	if _, ok := a.deployments[deploymentID]; !ok {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}
	a.nextID++
	token := vmcloudapi.AccessToken{
		ID:          fmt.Sprintf("token-%d", a.nextID),
		Secret:      fmt.Sprintf("secret-%d", a.nextID),
		Type:        req.Type,
		Description: req.Description,
		CreatedBy:   "test@example.com",
		CreatedAt:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		TenantID:    req.TenantID,
	}
	if a.tokens[deploymentID] == nil {
		a.tokens[deploymentID] = map[string]vmcloudapi.AccessToken{}
	}
	a.tokens[deploymentID][token.ID] = token
	writeJSON(w, http.StatusOK, token)
}

func (a *fakeAPI) getToken(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	token, ok := a.tokens[r.PathValue("id")][r.PathValue("token")]
	if !ok {
		writeError(w, http.StatusNotFound, "no such token %s", r.PathValue("token"))
		return
	}
	writeJSON(w, http.StatusOK, token)
}

func (a *fakeAPI) deleteToken(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	deploymentID, tokenID := r.PathValue("id"), r.PathValue("token")
	if _, ok := a.tokens[deploymentID][tokenID]; !ok {
		writeError(w, http.StatusNotFound, "no such token %s", tokenID)
		return
	}
	delete(a.tokens[deploymentID], tokenID)
	writeJSON(w, http.StatusOK, map[string]string{})
}

func (a *fakeAPI) getRuleFile(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	content, ok := a.ruleFiles[r.PathValue("id")][r.PathValue("name")]
	if !ok {
		writeError(w, http.StatusNotFound, "file %s not found", r.PathValue("name"))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, content)
}

func (a *fakeAPI) putRuleFile(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: %s", err)
		return
	}
	if !strings.Contains(string(body), "groups:") {
		writeError(w, http.StatusBadRequest, "invalid rule file")
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	deploymentID := r.PathValue("id")
	d, ok := a.deployments[deploymentID]
	if !ok {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}
	if d.info.Status != vmcloudapi.DeploymentStatusRunning {
		// The deployment moves on as if time passed between retries.
		d.advance()
		writeError(w, http.StatusLocked, "Cannot modify component params. Resource in provisioning state. Retry request later or create support ticket.")
		return
	}
	if a.ruleFiles[deploymentID] == nil {
		a.ruleFiles[deploymentID] = map[string]string{}
	}
	a.ruleFiles[deploymentID][r.PathValue("name")] = string(body)
	writeJSON(w, http.StatusOK, map[string]string{})
}

func (a *fakeAPI) deleteRuleFile(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	deploymentID, name := r.PathValue("id"), r.PathValue("name")
	if _, ok := a.ruleFiles[deploymentID][name]; !ok {
		writeError(w, http.StatusNotFound, "file %s not found", name)
		return
	}
	delete(a.ruleFiles[deploymentID], name)
	writeJSON(w, http.StatusOK, map[string]string{})
}
