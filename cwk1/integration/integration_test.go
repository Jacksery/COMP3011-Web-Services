package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

func waitForHealthy(t *testing.T) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	for i := 0; i < 30; i++ {
		resp, err := client.Get("http://localhost:8080/healthz")
		if err == nil && resp.StatusCode == http.StatusOK {
			if err := resp.Body.Close(); err != nil {
				t.Fatalf("failed to close response body: %v", err)
			}
			return
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatalf("service did not become healthy in time")
}

func TestIntegration_FullFlow(t *testing.T) {
	waitForHealthy(t)
	client := &http.Client{Timeout: 5 * time.Second}

	// login
	login := map[string]string{"username": "admin", "password": "password"}
	b, _ := json.Marshal(login)
	resp, err := client.Post("http://localhost:8080/auth/login", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("failed to close response body: %v", err)
		}
		t.Fatalf("login returned non-200: %d %s", resp.StatusCode, string(body))
	}
	var lr map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		if cerr := resp.Body.Close(); cerr != nil {
			t.Fatalf("failed to close response body: %v", cerr)
		}
		t.Fatalf("failed to parse login response: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("failed to close response body: %v", err)
	}
	tok, ok := lr["token"]
	if !ok || tok == "" {
		t.Fatalf("no token in login response")
	}

	// create product
	prod := map[string]interface{}{"product_id": "int-1", "product_name": "Integration"}
	pb, _ := json.Marshal(prod)
	req, _ := http.NewRequest(http.MethodPost, "http://localhost:8080/admin/products", bytes.NewReader(pb))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("create product request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("failed to close response body: %v", err)
		}
		t.Fatalf("create returned non-201: %d %s", resp.StatusCode, string(body))
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("failed to close response body: %v", err)
	}

	// get product
	resp, err = client.Get("http://localhost:8080/products/int-1")
	if err != nil {
		t.Fatalf("get product failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("failed to close response body: %v", err)
		}
		t.Fatalf("get returned non-200: %d %s", resp.StatusCode, string(body))
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("failed to close response body: %v", err)
	}

	// delete product
	req, _ = http.NewRequest(http.MethodDelete, "http://localhost:8080/admin/products/int-1", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("delete product failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("failed to close response body: %v", err)
		}
		t.Fatalf("delete returned non-200: %d %s", resp.StatusCode, string(body))
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("failed to close response body: %v", err)
	}
}
