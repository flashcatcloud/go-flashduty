package flashduty

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestReadPrometheusLabelValuesSendsPathAndHeader(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		if got := r.URL.EscapedPath(); got != "/monit/prometheus/api/v1/label/job%20name/values" {
			t.Errorf("escaped path = %s", got)
		}
		if got := r.URL.Query().Get("app_key"); got != "KEY" {
			t.Errorf("app_key = %q", got)
		}
		if got := r.Header.Get("X-DSID"); got != "42" {
			t.Errorf("X-DSID = %q", got)
		}
		w.Header().Set("Flashcat-Request-Id", "RIDL")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"status":"success","data":["api","db","worker"]}`)
	})

	out, resp, err := c.DataSources.ReadPrometheusLabelValues(context.Background(), 42, "job name")
	if err != nil {
		t.Fatalf("ReadPrometheusLabelValues error: %v", err)
	}
	if resp == nil || resp.StatusCode != http.StatusOK || resp.RequestID != "RIDL" {
		t.Fatalf("response meta = %+v", resp)
	}
	if out == nil || out.Status != "success" || len(out.Data) != 3 || out.Data[1] != "db" {
		t.Fatalf("decoded response = %+v", out)
	}
}

func TestReadPrometheusLabelValuesSurfacesNonEnvelopeErrorBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "missing header: X-DSID")
	})

	out, resp, err := c.DataSources.ReadPrometheusLabelValues(context.Background(), 42, "job")
	if err == nil {
		t.Fatal("expected an error")
	}
	if out != nil {
		t.Fatalf("expected nil result on error, got %+v", out)
	}
	if resp == nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("response meta = %+v", resp)
	}
	if got := err.Error(); got == "" {
		t.Fatal("expected a non-empty error message")
	}
}

func TestReadPrometheusLabelValuesValidatesInputs(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("request should not be sent")
	})

	if _, _, err := c.DataSources.ReadPrometheusLabelValues(context.Background(), 0, "job"); err == nil {
		t.Fatal("expected empty data source id error")
	}
	if _, _, err := c.DataSources.ReadPrometheusLabelValues(context.Background(), 42, ""); err == nil {
		t.Fatal("expected empty label name error")
	}
}
