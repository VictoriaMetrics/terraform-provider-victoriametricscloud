package provider

import (
	"errors"
	"net/http"
	"strings"

	vmcloudapi "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
)

// isNotFound reports whether err is an API response with status 404, which means
// the object was deleted outside Terraform.
func isNotFound(err error) bool {
	var apiErr *vmcloudapi.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// isNoChanges reports whether err is the API refusing an update because nothing it
// compares has changed. The API does not compare flags, so an update that changes
// only flags is answered this way too.
func isNoChanges(err error) bool {
	var apiErr *vmcloudapi.APIError
	return errors.As(err, &apiErr) &&
		apiErr.StatusCode == http.StatusBadRequest &&
		strings.Contains(strings.ToLower(apiErr.Message), "no changes")
}

// isLocked reports whether err is an API response with status 423, which the API
// sends while a deployment is being provisioned or updated.
func isLocked(err error) bool {
	var apiErr *vmcloudapi.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusLocked
}
