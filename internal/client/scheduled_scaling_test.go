// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type scheduledScalingEndpointTest struct {
	name        string
	path        string
	body        map[string]any
	getResponse string
	wantGet     any
	get         func(*SssClient, string) (any, error)
	create      func(*SssClient, string) error
	update      func(*SssClient, string) error
	delete      func(*SssClient, string) error
}

func TestScheduledScalingClientRequests(t *testing.T) {
	serviceID := "group/name with space"
	endpoints := scheduledScalingEndpointTests()

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				wantPath := endpoint.path + "/group%2Fname%20with%20space"
				if endpoint.name == "ecs" {
					// Keep the ECS client's existing path escaping behavior unchanged.
					wantPath = endpoint.path + "/group%252Fname%2520with%2520space"
				}
				if got := r.URL.EscapedPath(); got != wantPath {
					t.Errorf("path = %q, want %q", got, wantPath)
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
					_, _ = w.Write([]byte(endpoint.getResponse))
				case http.MethodPost, http.MethodPut:
					if got, want := r.Header.Get("Content-Type"), "application/json"; got != want {
						t.Errorf("Content-Type = %q, want %q", got, want)
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("decode request body: %v", err)
					}
					if !reflect.DeepEqual(body, endpoint.body) {
						t.Errorf("body = %#v, want %#v", body, endpoint.body)
					}
					if r.Method == http.MethodPost {
						w.WriteHeader(http.StatusCreated)
					}
				case http.MethodDelete:
					w.WriteHeader(http.StatusOK)
				default:
					t.Errorf("method = %q", r.Method)
				}
			}))
			defer server.Close()

			client := testSssClient(server)
			gotGet, err := endpoint.get(client, serviceID)
			if err != nil {
				t.Fatalf("GET: %v", err)
			}
			if !reflect.DeepEqual(gotGet, endpoint.wantGet) {
				t.Errorf("GET response = %#v, want %#v", gotGet, endpoint.wantGet)
			}
			if err := endpoint.create(client, serviceID); err != nil {
				t.Fatalf("POST: %v", err)
			}
			if err := endpoint.update(client, serviceID); err != nil {
				t.Fatalf("PUT: %v", err)
			}
			if err := endpoint.delete(client, serviceID); err != nil {
				t.Fatalf("DELETE: %v", err)
			}
		})
	}
}

func TestScheduledScalingClient422Errors(t *testing.T) {
	for _, endpoint := range scheduledScalingEndpointTests() {
		if endpoint.name == "ecs" {
			// ECS keeps its existing status-only error handling; this test covers
			// the newer scheduled-scaling clients that parse structured details.
			continue
		}
		t.Run(endpoint.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(`{"detail":"invalid capacity","errors":[{"location":"capacity.low","message":"must be valid","value":"secret"}]}`))
			}))
			defer server.Close()

			err := endpoint.create(testSssClient(server), "service")
			if err == nil {
				t.Fatal("expected error")
			}
			if got := err.Error(); !strings.Contains(got, "422 Unprocessable Entity") || !strings.Contains(got, "invalid capacity") || !strings.Contains(got, "capacity.low: must be valid") {
				t.Errorf("error = %q, missing 422 details", got)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Errorf("error = %q, exposes error value", err)
			}
		})
	}
}

func TestScheduledScalingClientMalformed422Fallback(t *testing.T) {
	for _, responseBody := range []string{"", `{"detail":`} {
		for _, endpoint := range scheduledScalingEndpointTests() {
			t.Run(endpoint.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnprocessableEntity)
					_, _ = w.Write([]byte(responseBody))
				}))
				defer server.Close()

				err := endpoint.create(testSssClient(server), "service")
				if err == nil || !strings.Contains(err.Error(), "422 Unprocessable Entity") {
					t.Errorf("error = %v, want status fallback", err)
				}
			})
		}
	}
}

func testSssClient(server *httptest.Server) *SssClient {
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		panic(err)
	}
	return NewSssClient(serverURL.Host, serverURL.Scheme, "user", "pass")
}

func scheduledScalingEndpointTests() []scheduledScalingEndpointTest {
	return []scheduledScalingEndpointTest{
		{
			name: "ecs",
			path: "/api/v1/services/ecs",
			body: map[string]any{
				"region": "eu-west-1", "minLowCapacity": float64(4), "minMediumCapacity": float64(10), "minHighCapacity": float64(18), "minExtremeCapacity": float64(18),
				"scaleUpTasksPerMinute": float64(2), "scaleUpLeadTimeMinutes": float64(7),
			},
			getResponse: `{"Name":"group/name with space","Region":"eu-west-1","MinLowCapacity":4,"MinMediumCapacity":10,"MinHighCapacity":18,"MinExtremeCapacity":18,"ScaleUpTasksPerMinute":2,"ScaleUpLeadTimeMinutes":7}`,
			wantGet: &EcsServiceResponse{
				Name: "group/name with space", Region: "eu-west-1", MinLowCapacity: 4, MinMediumCapacity: 10, MinHighCapacity: 18, MinExtremeCapacity: 18,
				ScaleUpTasksPerMinute: 2, ScaleUpLeadTimeMinutes: 7,
			},
			get: func(c *SssClient, id string) (any, error) { return c.GetEcsService(id) },
			create: func(c *SssClient, id string) error {
				return c.CreateEcsService(id, EcsServicePostBody{Region: "eu-west-1", MinLowCapacity: 4, MinMediumCapacity: 10, MinHighCapacity: 18, MinExtremeCapacity: 18, ScaleUpTasksPerMinute: 2, ScaleUpLeadTimeMinutes: 7})
			},
			update: func(c *SssClient, id string) error {
				return c.UpdateEcsService(id, EcsServicePostBody{Region: "eu-west-1", MinLowCapacity: 4, MinMediumCapacity: 10, MinHighCapacity: 18, MinExtremeCapacity: 18, ScaleUpTasksPerMinute: 2, ScaleUpLeadTimeMinutes: 7})
			},
			delete: func(c *SssClient, id string) error {
				_, err := c.DeleteEcsService(id)
				return err
			},
		},
		{
			name: "valkey replicas",
			path: "/api/v1/services/valkey-replicas",
			body: map[string]any{
				"region": "eu-west-1", "scaleUpLeadTimeMinutes": float64(60), "replicaCountLow": float64(0), "replicaCountMedium": float64(1), "replicaCountHigh": float64(2), "replicaCountExtreme": float64(3),
			},
			getResponse: `{"serviceId":"group/name with space","region":"eu-west-1","scaleUpLeadTimeMinutes":60,"replicaCountLow":0,"replicaCountMedium":1,"replicaCountHigh":2,"replicaCountExtreme":3,"$schema":"ignored"}`,
			wantGet:     &ValkeyReplicaScalingResponse{ServiceID: "group/name with space", Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60, ReplicaCountLow: 0, ReplicaCountMedium: 1, ReplicaCountHigh: 2, ReplicaCountExtreme: 3},
			get: func(c *SssClient, id string) (any, error) {
				return c.GetValkeyReplicaScaling(id)
			},
			create: func(c *SssClient, id string) error {
				return c.CreateValkeyReplicaScaling(id, ValkeyReplicaScalingPostBody{Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60, ReplicaCountLow: 0, ReplicaCountMedium: 1, ReplicaCountHigh: 2, ReplicaCountExtreme: 3})
			},
			update: func(c *SssClient, id string) error {
				return c.UpdateValkeyReplicaScaling(id, ValkeyReplicaScalingPostBody{Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60, ReplicaCountLow: 0, ReplicaCountMedium: 1, ReplicaCountHigh: 2, ReplicaCountExtreme: 3})
			},
			delete: func(c *SssClient, id string) error { return c.DeleteValkeyReplicaScaling(id) },
		},
		{
			name: "valkey shards",
			path: "/api/v1/services/valkey-shards",
			body: map[string]any{
				"region": "eu-west-1", "scaleUpLeadTimeMinutes": float64(60),
				"lowCapacity": map[string]any{"minShardCount": float64(11), "maxShardCount": float64(12)}, "mediumCapacity": map[string]any{"minShardCount": float64(21), "maxShardCount": float64(22)}, "highCapacity": map[string]any{"minShardCount": float64(31), "maxShardCount": float64(32)}, "extremeCapacity": map[string]any{"minShardCount": float64(41), "maxShardCount": float64(42)},
			},
			getResponse: `{"serviceId":"group/name with space","region":"eu-west-1","scaleUpLeadTimeMinutes":60,"lowCapacity":{"minShardCount":11,"maxShardCount":12},"mediumCapacity":{"minShardCount":21,"maxShardCount":22},"highCapacity":{"minShardCount":31,"maxShardCount":32},"extremeCapacity":{"minShardCount":41,"maxShardCount":42},"$schema":"ignored"}`,
			wantGet: &ValkeyShardScalingResponse{
				ServiceID: "group/name with space", Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60,
				LowCapacity: ValkeyShardCapacity{MinShardCount: 11, MaxShardCount: 12}, MediumCapacity: ValkeyShardCapacity{MinShardCount: 21, MaxShardCount: 22}, HighCapacity: ValkeyShardCapacity{MinShardCount: 31, MaxShardCount: 32}, ExtremeCapacity: ValkeyShardCapacity{MinShardCount: 41, MaxShardCount: 42},
			},
			get: func(c *SssClient, id string) (any, error) {
				return c.GetValkeyShardScaling(id)
			},
			create: func(c *SssClient, id string) error { return c.CreateValkeyShardScaling(id, valkeyShardTestBody()) },
			update: func(c *SssClient, id string) error { return c.UpdateValkeyShardScaling(id, valkeyShardTestBody()) },
			delete: func(c *SssClient, id string) error { return c.DeleteValkeyShardScaling(id) },
		},
		{
			name: "aurora readers",
			path: "/api/v1/services/aurora-readers",
			body: map[string]any{
				"region": "eu-west-1", "scaleUpLeadTimeMinutes": float64(60),
				"lowCapacity": map[string]any{"minReaders": float64(1), "maxReaders": float64(2)}, "mediumCapacity": map[string]any{"minReaders": float64(2), "maxReaders": float64(3)}, "highCapacity": map[string]any{"minReaders": float64(3), "maxReaders": float64(4)}, "extremeCapacity": map[string]any{"minReaders": float64(4), "maxReaders": float64(5)},
			},
			getResponse: `{"serviceId":"group/name with space","region":"eu-west-1","scaleUpLeadTimeMinutes":60,"lowCapacity":{"minReaders":1,"maxReaders":2},"mediumCapacity":{"minReaders":2,"maxReaders":3},"highCapacity":{"minReaders":3,"maxReaders":4},"extremeCapacity":{"minReaders":4,"maxReaders":5},"$schema":"ignored"}`,
			wantGet: &AuroraReaderScalingResponse{
				ServiceID: "group/name with space", Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60,
				LowCapacity: AuroraReaderCapacity{MinReaders: 1, MaxReaders: 2}, MediumCapacity: AuroraReaderCapacity{MinReaders: 2, MaxReaders: 3}, HighCapacity: AuroraReaderCapacity{MinReaders: 3, MaxReaders: 4}, ExtremeCapacity: AuroraReaderCapacity{MinReaders: 4, MaxReaders: 5},
			},
			get: func(c *SssClient, id string) (any, error) {
				return c.GetAuroraReaderScaling(id)
			},
			create: func(c *SssClient, id string) error { return c.CreateAuroraReaderScaling(id, auroraReaderTestBody()) },
			update: func(c *SssClient, id string) error { return c.UpdateAuroraReaderScaling(id, auroraReaderTestBody()) },
			delete: func(c *SssClient, id string) error { return c.DeleteAuroraReaderScaling(id) },
		},
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

func valkeyShardTestBody() ValkeyShardScalingPostBody {
	return ValkeyShardScalingPostBody{
		Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60,
		LowCapacity: ValkeyShardCapacity{MinShardCount: 11, MaxShardCount: 12}, MediumCapacity: ValkeyShardCapacity{MinShardCount: 21, MaxShardCount: 22}, HighCapacity: ValkeyShardCapacity{MinShardCount: 31, MaxShardCount: 32}, ExtremeCapacity: ValkeyShardCapacity{MinShardCount: 41, MaxShardCount: 42},
	}
}

func auroraReaderTestBody() AuroraReaderScalingPostBody {
	return AuroraReaderScalingPostBody{
		Region: "eu-west-1", ScaleUpLeadTimeMinutes: 60,
		LowCapacity: AuroraReaderCapacity{MinReaders: 1, MaxReaders: 2}, MediumCapacity: AuroraReaderCapacity{MinReaders: 2, MaxReaders: 3}, HighCapacity: AuroraReaderCapacity{MinReaders: 3, MaxReaders: 4}, ExtremeCapacity: AuroraReaderCapacity{MinReaders: 4, MaxReaders: 5},
	}
}
