// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestEcsClientRequests(t *testing.T) {
	serviceID := "group/name with space"
	wantBody := map[string]any{
		"region": "eu-west-1", "minLowCapacity": float64(4), "minMediumCapacity": float64(10), "minHighCapacity": float64(18), "minExtremeCapacity": float64(18),
		"scaleUpTasksPerMinute": float64(2), "scaleUpLeadTimeMinutes": float64(7),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The legacy ECS client double-escapes IDs; preserve its existing behavior.
		if got, want := r.URL.EscapedPath(), "/api/v1/services/ecs/group%252Fname%2520with%2520space"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "user" || password != "pass" {
			t.Errorf("Basic authentication = %q/%q, want user/pass", username, password)
		}
		if got, want := r.Header.Get("Accept"), "application/json, application/problem+json"; got != want {
			t.Errorf("Accept = %q, want %q", got, want)
		}
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Name":"group/name with space","Region":"eu-west-1","MinLowCapacity":4,"MinMediumCapacity":10,"MinHighCapacity":18,"MinExtremeCapacity":18,"ScaleUpTasksPerMinute":2,"ScaleUpLeadTimeMinutes":7}`))
		case http.MethodPost, http.MethodPut:
			if got, want := r.Header.Get("Content-Type"), "application/json"; got != want {
				t.Errorf("Content-Type = %q, want %q", got, want)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode request body: %v", err)
				return
			}
			if !reflect.DeepEqual(body, wantBody) {
				t.Errorf("body = %#v, want %#v", body, wantBody)
			}
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
			}
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %q", r.Method)
		}
	}))
	defer server.Close()

	client := testSssClient(server)
	response, err := client.GetEcsService(serviceID)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	wantResponse := &EcsServiceResponse{
		Name: serviceID, Region: "eu-west-1", MinLowCapacity: 4, MinMediumCapacity: 10, MinHighCapacity: 18, MinExtremeCapacity: 18,
		ScaleUpTasksPerMinute: 2, ScaleUpLeadTimeMinutes: 7,
	}
	if !reflect.DeepEqual(response, wantResponse) {
		t.Errorf("GET response = %#v, want %#v", response, wantResponse)
	}
	body := EcsServicePostBody{
		Region: "eu-west-1", MinLowCapacity: 4, MinMediumCapacity: 10, MinHighCapacity: 18, MinExtremeCapacity: 18,
		ScaleUpTasksPerMinute: 2, ScaleUpLeadTimeMinutes: 7,
	}
	if err := client.CreateEcsService(serviceID, body); err != nil {
		t.Fatalf("POST: %v", err)
	}
	if err := client.UpdateEcsService(serviceID, body); err != nil {
		t.Fatalf("PUT: %v", err)
	}
	if _, err := client.DeleteEcsService(serviceID); err != nil {
		t.Fatalf("DELETE: %v", err)
	}
}

func TestEcsClientWritesExplicitZeroScaleUpFieldsAndReadsLegacyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode request body: %v", err)
				return
			}
			for _, field := range []string{"scaleUpTasksPerMinute", "scaleUpLeadTimeMinutes"} {
				value, exists := body[field]
				if !exists || value != float64(0) {
					t.Errorf("%s = %#v (present %t), want explicit zero", field, value, exists)
				}
			}
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
			}
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Name":"service","Region":"eu-west-1","MinLowCapacity":1,"MinMediumCapacity":2,"MinHighCapacity":3,"MinExtremeCapacity":4}`))
		default:
			t.Errorf("unexpected method %q", r.Method)
		}
	}))
	defer server.Close()

	client := testSssClient(server)
	body := EcsServicePostBody{Region: "eu-west-1", MinLowCapacity: 1, MinMediumCapacity: 2, MinHighCapacity: 3, MinExtremeCapacity: 4}
	if err := client.CreateEcsService("service", body); err != nil {
		t.Fatalf("POST: %v", err)
	}
	if err := client.UpdateEcsService("service", body); err != nil {
		t.Fatalf("PUT: %v", err)
	}
	response, err := client.GetEcsService("service")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	if response.ScaleUpTasksPerMinute != 0 || response.ScaleUpLeadTimeMinutes != 0 {
		t.Errorf("legacy response scale-up values = %d/%d, want 0/0", response.ScaleUpTasksPerMinute, response.ScaleUpLeadTimeMinutes)
	}
}

func TestEcsClient422StatusOnly(t *testing.T) {
	for _, responseBody := range []string{"", `{"detail":`, `{"detail":"invalid capacity"}`} {
		t.Run(responseBody, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(responseBody))
			}))
			defer server.Close()

			err := testSssClient(server).CreateEcsService("service", EcsServicePostBody{})
			if err == nil || !strings.Contains(err.Error(), "422 Unprocessable Entity") {
				t.Fatalf("error = %v, want status-only 422", err)
			}
			if strings.Contains(err.Error(), "invalid capacity") {
				t.Errorf("error = %q, should not include response details", err)
			}
		})
	}
}
