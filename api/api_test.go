package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	types "github.com/elrizwiraswara/godwit/types"
)

const testKey = "test-key"

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func post(t *testing.T, h http.Handler, path, key string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func samplePair(t *testing.T, n string) generateRequest {
	return generateRequest{
		SourceSampleCSV: readFile(t, "../input/samples/source_sample_data_"+n+".csv"),
		TargetSampleCSV: readFile(t, "../input/samples/target_sample_data_"+n+".csv"),
	}
}

func findTarget(schema []types.ColumnSchema, target string) types.ColumnSchema {
	for _, c := range schema {
		if c.TargetColumn == target {
			return c
		}
	}
	return types.ColumnSchema{}
}

func TestAuth(t *testing.T) {
	h := New(Config{APIKeys: []string{testKey}})
	req := samplePair(t, "2")

	for _, key := range []string{"", "wrong-key"} {
		if rec := post(t, h, "/v1/schemas", key, req); rec.Code != http.StatusUnauthorized {
			t.Errorf("key %q: status %d, want 401", key, rec.Code)
		}
	}
	if rec := post(t, h, "/v1/schemas", testKey, req); rec.Code != http.StatusOK {
		t.Errorf("valid key: status %d: %s", rec.Code, rec.Body)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("healthz: status %d", rec.Code)
	}

	for key, want := range map[string]int{"": http.StatusUnauthorized, "wrong-key": http.StatusUnauthorized, testKey: http.StatusOK} {
		req := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Errorf("status with key %q: %d, want %d", key, rec.Code, want)
		}
	}
}

func TestGenerateThenConvert(t *testing.T) {
	h := New(Config{APIKeys: []string{testKey}})

	rec := post(t, h, "/v1/schemas", testKey, samplePair(t, "2"))
	if rec.Code != http.StatusOK {
		t.Fatalf("generate: status %d: %s", rec.Code, rec.Body)
	}
	var gen generateResponse
	json.Unmarshal(rec.Body.Bytes(), &gen)

	status := findTarget(gen.SourceSchema, "stock_status")
	if status.Column != "availability" || status.ValuesMapping["L"] != "low_stock" {
		t.Fatalf("stock_status mapping = %+v", status)
	}
	if len(gen.SourceColumns) != 10 {
		t.Errorf("source_columns = %v", gen.SourceColumns)
	}

	rec = post(t, h, "/v1/convert", testKey, convertRequest{
		SourceCSV:    readFile(t, "../input/source_data_2.csv"),
		SourceSchema: gen.SourceSchema,
		TargetSchema: gen.TargetSchema,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("convert: status %d: %s", rec.Code, rec.Body)
	}
	var conv convertResponse
	json.Unmarshal(rec.Body.Bytes(), &conv)
	if conv.Rows != 50 || len(conv.Unmapped) != 0 {
		t.Errorf("rows=%d unmapped=%v", conv.Rows, conv.Unmapped)
	}
	if n := strings.Count(conv.CSV, ",low_stock,"); n != 3 {
		t.Errorf("low_stock rows = %d, want 3", n)
	}
}

func TestOverrides(t *testing.T) {
	h := New(Config{})

	req := samplePair(t, "2")
	req.Overrides = map[string]string{"stock_status": "", "warehouse_name": "storage_loc_id", "warehouse_id": "storage_location"}
	rec := post(t, h, "/v1/schemas", "", req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var gen generateResponse
	json.Unmarshal(rec.Body.Bytes(), &gen)
	if c := findTarget(gen.SourceSchema, "stock_status"); c.Column != "" || c.Evidence != "left unmatched manually" {
		t.Errorf("stock_status = %+v", c)
	}
	if c := findTarget(gen.SourceSchema, "warehouse_name"); c.Column != "storage_loc_id" {
		t.Errorf("warehouse_name = %+v", c)
	}

	req.Overrides = map[string]string{"no_such_column": "availability"}
	if rec := post(t, h, "/v1/schemas", "", req); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown column: status %d, want 400", rec.Code)
	}
}

func TestLimits(t *testing.T) {
	h := New(Config{MaxBodyBytes: 1024, MaxSampleRows: 5})

	if rec := post(t, h, "/v1/schemas", "", samplePair(t, "2")); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("large body: status %d, want 413", rec.Code)
	}

	small := generateRequest{
		SourceSampleCSV: "a,b\n1,x\n2,y\n3,x\n4,y\n5,x\n6,y\n",
		TargetSampleCSV: "a,b\n1,x\n",
	}
	if rec := post(t, h, "/v1/schemas", "", small); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "allowed for a sample") {
		t.Errorf("too many rows: status %d: %s", rec.Code, rec.Body)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/schemas", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("content type: status %d, want 415", rec.Code)
	}
}

func TestCORS(t *testing.T) {
	h := New(Config{APIKeys: []string{testKey}, CORSOrigins: []string{"https://app.example.com"}})

	for origin, allowed := range map[string]bool{"https://app.example.com": true, "https://evil.example.com": false} {
		req := httptest.NewRequest(http.MethodOptions, "/v1/schemas", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		got := rec.Header().Get("Access-Control-Allow-Origin") == origin
		if got != allowed {
			t.Errorf("origin %s: allowed=%v, want %v", origin, got, allowed)
		}
		if allowed && rec.Code != http.StatusNoContent {
			t.Errorf("preflight status %d, want 204", rec.Code)
		}
	}
}
