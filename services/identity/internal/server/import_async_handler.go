package server

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	ggidtenant "github.com/ggid/ggid/pkg/tenant"
)

// handleImportAsync creates a new async import job.
// POST /api/v1/identity/users/import-async
//
// Supports multipart file upload or inline JSON body.
// Returns job_id immediately; processing runs asynchronously.
func (h *HTTPHandler) handleImportAsync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.importJobRepo == nil {
		writeError(w, http.StatusServiceUnavailable, "import job system not configured")
		return
	}

	// Resolve tenant.
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}

	// Parse the uploaded file or inline JSON.
	var records []ImportUserRecord
	var format string

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		// Multipart file upload.
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeError(w, http.StatusBadRequest, "failed to parse multipart form")
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "file field required")
			return
		}
		defer file.Close()

		format = detectFormat(header.Filename)
		data, err := io.ReadAll(file)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read file")
			return
		}

		records, err = parseRecords(data, format)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	} else {
		// Inline JSON body.
		format = "json"
		if err := json.NewDecoder(r.Body).Decode(&records); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	if len(records) == 0 {
		writeError(w, http.StatusBadRequest, "no records to import")
		return
	}

	// Create the job.
	now := time.Now().UTC()
	job := &ImportJob{
		ID:        "imp-" + uuid.New().String(),
		TenantID:  tc.TenantID,
		Format:    format,
		Status:    "pending",
		Total:     len(records),
		CreatedAt: now,
	}

	if err := h.importJobRepo.Create(r.Context(), job); err != nil {
		slog.Error("failed to create import job", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create import job")
		return
	}

	// Process asynchronously.
	go h.ProcessImportRecords(r.Context(), job.ID, tc.TenantID, records)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"job_id":  job.ID,
		"status":  "pending",
		"total":   job.Total,
		"format":  job.Format,
		"message": "import job created; poll GET /api/v1/identity/users/import-async/" + job.ID + " for status",
	})
}

// handleImportAsyncStatus returns the current status of an import job.
// GET /api/v1/identity/users/import-async/:job_id
func (h *HTTPHandler) handleImportAsyncStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.importJobRepo == nil {
		writeError(w, http.StatusServiceUnavailable, "import job system not configured")
		return
	}

	jobID := strings.TrimPrefix(r.URL.Path, "/api/v1/identity/users/import-async/")
	// Remove any trailing slash or query.
	jobID = strings.TrimSuffix(jobID, "/")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "job_id required in path")
		return
	}

	job, err := h.importJobRepo.Get(r.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	writeJSON(w, http.StatusOK, job)
}

// handleImportAsyncList lists all import jobs for the tenant.
// GET /api/v1/identity/users/import-async
func (h *HTTPHandler) handleImportAsyncList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.importJobRepo == nil {
		writeError(w, http.StatusServiceUnavailable, "import job system not configured")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}

	jobs, err := h.importJobRepo.List(r.Context(), tc.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list jobs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

// detectFormat infers the file format from the filename extension.
func detectFormat(filename string) string {
	if strings.HasSuffix(strings.ToLower(filename), ".csv") {
		return "csv"
	}
	return "json"
}

// parseRecords parses user records from JSON or CSV data.
func parseRecords(data []byte, format string) ([]ImportUserRecord, error) {
	switch format {
	case "csv":
		return parseCSVRecords(data)
	default:
		return parseJSONRecords(data)
	}
}

func parseJSONRecords(data []byte) ([]ImportUserRecord, error) {
	var records []ImportUserRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func parseCSVRecords(data []byte) ([]ImportUserRecord, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1 // Allow variable fields

	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	// Build column index map.
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	var records []ImportUserRecord
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		rec := ImportUserRecord{}
		if idx, ok := colMap["username"]; ok && idx < len(row) {
			rec.Username = row[idx]
		}
		if idx, ok := colMap["email"]; ok && idx < len(row) {
			rec.Email = row[idx]
		}
		if idx, ok := colMap["password"]; ok && idx < len(row) {
			rec.Password = row[idx]
		}
		if idx, ok := colMap["display_name"]; ok && idx < len(row) {
			rec.DisplayName = row[idx]
		}
		records = append(records, rec)
	}

	return records, nil
}
