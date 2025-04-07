// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
)

type SssClient struct {
	host         string
	protocol     string
	authUsername string
	authPassword string
	httpClient   *http.Client
}

// NewSssClient creates a new client for the Scheduled Scaling Service API.
func NewSssClient(host string, protocol string, authUsername string, authPassword string) *SssClient {
	httpClient := &http.Client{}

	return &SssClient{
		host:         host,
		protocol:     protocol,
		authUsername: authUsername,
		authPassword: authPassword,
		httpClient:   httpClient,
	}
}

type scalableType string

const scalableTypeECS scalableType = "ecs"
const scalableTypeDynamoDB scalableType = "dynamodbtable"

func getOrDeleteScalable[T any](client *SssClient, scalableType scalableType, scalableId string, method string) (*T, error) {
	url := url.URL{
		Scheme: client.protocol,
		Host:   client.host,
		Path:   path.Join("/api/v1/services/", string(scalableType), url.PathEscape(scalableId)),
	}
	req, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(client.authUsername, client.authPassword)
	req.Header.Set("Accept", "application/json, application/problem+json")
	response, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if method == "DELETE" {
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to delete scalable %s/%s: %s", string(scalableType), scalableId, response.Status)
		}
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get scalable %s/%s: %s", string(scalableType), scalableId, response.Status)
	}
	// Parse the response body into a EcsServiceResponse struct.
	var scalableResponse T
	err = json.NewDecoder(response.Body).Decode(&scalableResponse)
	if err != nil {
		return nil, err
	}
	return &scalableResponse, nil
}

func editScalable[T any](client *SssClient, scalableType scalableType, scalableId string, capacities T, method string) error {
	url := url.URL{
		Scheme: client.protocol,
		Host:   client.host,
		Path:   path.Join("/api/v1/services/", string(scalableType), url.PathEscape(scalableId)),
	}

	body, err := json.Marshal(capacities)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, url.String(), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(client.authUsername, client.authPassword)
	req.Header.Set("Accept", "application/json, application/problem+json")
	req.Header.Set("Content-Type", "application/json")
	response, err := client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if method == "POST" && response.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create service %s/%s: %s", scalableType, scalableId, response.Status)
	}
	if method == "PUT" && response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update service %s/%s: %s", scalableType, scalableId, response.Status)
	}
	return nil
}
