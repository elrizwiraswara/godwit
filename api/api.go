// Package api serves schema generation and CSV conversion over HTTP.
//
// Uploaded CSV data is processed in memory and never stored or logged; only
// the method, path, status and duration of each request are logged.
package api

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"slices"
	"strings"
	"time"

	convert "github.com/elrizwiraswara/godwit/convert"
	infer "github.com/elrizwiraswara/godwit/infer"
	types "github.com/elrizwiraswara/godwit/types"
	utils "github.com/elrizwiraswara/godwit/utils"
)

type Config struct {
	// APIKeys are accepted as "Authorization: Bearer <key>". Empty disables
	// authentication; the server only allows that on localhost.
	APIKeys []string
	// Embedder enables step 4 of schema generation; nil runs steps 1-3
	Embedder infer.Embedder
	// MaxBodyBytes limits request size (default 32 MB)
	MaxBodyBytes int64
	// MaxSampleRows and MaxColumns limit schema generation input, which is
	// meant for samples (defaults 5000 and 500)
	MaxSampleRows int
	MaxColumns    int
	// MaxConcurrent limits simultaneous schema generations (default: CPUs)
	MaxConcurrent int
	// CORSOrigins lists browser origins allowed to call the API (Flutter web)
	CORSOrigins []string
}

type server struct {
	cfg Config
	sem chan struct{}
}

func New(cfg Config) http.Handler {
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = 32 << 20
	}
	if cfg.MaxSampleRows <= 0 {
		cfg.MaxSampleRows = 5000
	}
	if cfg.MaxColumns <= 0 {
		cfg.MaxColumns = 500
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = runtime.NumCPU()
	}
	s := &server{cfg: cfg, sem: make(chan struct{}, cfg.MaxConcurrent)}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	// Same as /healthz, but requires a valid key, so clients can check theirs
	mux.Handle("GET /v1/status", s.authorized(http.HandlerFunc(s.health)))
	mux.Handle("POST /v1/schemas", s.authorized(http.HandlerFunc(s.generateSchemas)))
	mux.Handle("POST /v1/convert", s.authorized(http.HandlerFunc(s.convert)))

	return logRequests(s.cors(mux))
}

type generateRequest struct {
	SourceSampleCSV string `json:"source_sample_csv"`
	TargetSampleCSV string `json:"target_sample_csv"`
	// Overrides maps target column -> source column ("" for none)
	Overrides map[string]string `json:"overrides,omitempty"`
}

type generateResponse struct {
	TargetSchema        []types.ColumnSchema `json:"target_schema"`
	SourceSchema        []types.ColumnSchema `json:"source_schema"`
	SourceColumns       []string             `json:"source_columns"`
	SharedKey           string               `json:"shared_key,omitempty"`
	UnusedSourceColumns []string             `json:"unused_source_columns"`
	EmbeddingModel      bool                 `json:"embedding_model"`
}

type convertRequest struct {
	SourceCSV    string               `json:"source_csv"`
	SourceSchema []types.ColumnSchema `json:"source_schema"`
	TargetSchema []types.ColumnSchema `json:"target_schema"`
}

type convertResponse struct {
	CSV      string             `json:"csv"`
	Rows     int                `json:"rows"`
	Unmapped []convert.Unmapped `json:"unmapped"`
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "embedding_model": s.cfg.Embedder != nil})
}

func (s *server) generateSchemas(w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if !s.decode(w, r, &req) {
		return
	}

	source, err := s.parseSample("source_sample_csv", req.SourceSampleCSV)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	target, err := s.parseSample("target_sample_csv", req.TargetSampleCSV)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Generation uses every CPU core, so run a limited number at once
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-r.Context().Done():
		return
	}

	result, err := infer.Generate(source, target, infer.Options{Embedder: s.cfg.Embedder, Overrides: req.Overrides})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, generateResponse{
		TargetSchema:        result.Target,
		SourceSchema:        result.Source,
		SourceColumns:       result.SourceColumns,
		SharedKey:           result.SharedKey,
		UnusedSourceColumns: nonNil(result.UnusedSource),
		EmbeddingModel:      s.cfg.Embedder != nil,
	})
}

func (s *server) convert(w http.ResponseWriter, r *http.Request) {
	var req convertRequest
	if !s.decode(w, r, &req) {
		return
	}
	if len(req.SourceSchema) == 0 || len(req.TargetSchema) == 0 {
		writeError(w, http.StatusBadRequest, "source_schema and target_schema are required")
		return
	}

	records, err := utils.ParseCSV(strings.NewReader(req.SourceCSV))
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("source_csv: %v", err))
		return
	}
	result, err := convert.Convert(records, req.SourceSchema, req.TargetSchema)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("source_csv: %v", err))
		return
	}

	var out bytes.Buffer
	writer := csv.NewWriter(&out)
	if err := writer.WriteAll(result.Records); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write CSV")
		return
	}

	writeJSON(w, http.StatusOK, convertResponse{
		CSV:      out.String(),
		Rows:     len(result.Records) - 1,
		Unmapped: result.Unmapped,
	})
}

func (s *server) parseSample(field, text string) ([][]string, error) {
	records, err := utils.ParseCSV(strings.NewReader(text))
	if err != nil {
		return nil, fmt.Errorf("%s: %v", field, err)
	}
	switch {
	case len(records) < 2:
		return nil, fmt.Errorf("%s: needs a header and at least one data row", field)
	case len(records)-1 > s.cfg.MaxSampleRows:
		return nil, fmt.Errorf("%s: %d rows is more than the %d allowed for a sample; send a representative subset", field, len(records)-1, s.cfg.MaxSampleRows)
	case len(records[0]) > s.cfg.MaxColumns:
		return nil, fmt.Errorf("%s: more than %d columns", field, s.cfg.MaxColumns)
	}
	return records, nil
}

// decode reads a JSON body, writing the error response itself on failure.
func (s *server) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}

	body := http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes)
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("request body is larger than %d bytes", s.cfg.MaxBodyBytes))
			return false
		}
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return false
	}
	return true
}

func (s *server) authorized(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(s.cfg.APIKeys) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if ok {
			for _, key := range s.cfg.APIKeys {
				if subtle.ConstantTimeCompare([]byte(token), []byte(key)) == 1 {
					next.ServeHTTP(w, r)
					return
				}
			}
		}
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeError(w, http.StatusUnauthorized, "missing or invalid API key")
	})
}

func (s *server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && slices.Contains(s.cfg.CORSOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Add("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if r.Context().Err() == context.Canceled {
			rec.status = 499 // client went away
		}
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Millisecond))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
