package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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

func (client *SssClient) getOrDeleteEcsService(serviceName string, method string) (*EcsServiceResponse, error) {
	tflog.Info(context.Background(), fmt.Sprintf("createOrDestroyEcsService %s %s", method, serviceName))
	url := url.URL{
		Scheme: client.protocol,
		Host:   client.host,
		Path:   path.Join("/api/v1/services/ecs", url.PathEscape(serviceName)),
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
	// Parse the response body into a EcsServiceResponse struct.
	var ecsServiceResponse EcsServiceResponse
	err = json.NewDecoder(response.Body).Decode(&ecsServiceResponse)
	tflog.Info(context.Background(), fmt.Sprintf("ecsServiceResponse: %v", ecsServiceResponse))
	if err != nil {
		return nil, err
	}
	return &ecsServiceResponse, nil
}

func (client *SssClient) editEcsService(serviceName string, capacities EcsServicePostBody, method string) (*EcsServiceResponse, error) {
	tflog.Info(context.Background(), fmt.Sprintf("editEcsService %s %s: %v", method, serviceName, capacities))
	url := url.URL{
		Scheme: client.protocol,
		Host:   client.host,
		Path:   path.Join("/api/v1/services/ecs", url.PathEscape(serviceName)),
	}

	body, err := json.Marshal(capacities)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, url.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(client.authUsername, client.authPassword)
	req.Header.Set("Accept", "application/json, application/problem+json")
	req.Header.Set("Content-Type", "application/json")
	response, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	// Parse the response body into a EcsServiceResponse struct.
	var ecsServiceResponse EcsServiceResponse
	err = json.NewDecoder(response.Body).Decode(&ecsServiceResponse)
	if err != nil {
		return nil, err
	}
	tflog.Info(context.Background(), fmt.Sprintf("ecsServiceResponse: %v", ecsServiceResponse))
	return &ecsServiceResponse, nil
}

func (client *SssClient) GetEcsService(serviceName string) (*EcsServiceResponse, error) {
	return client.getOrDeleteEcsService(serviceName, "GET")
}

func (client *SssClient) CreateEcsService(serviceName string, capacities EcsServicePostBody) (*EcsServiceResponse, error) {
	return client.editEcsService(serviceName, capacities, "POST")
}

func (client *SssClient) UpdateEcsService(serviceName string, capacities EcsServicePostBody) (*EcsServiceResponse, error) {
	return client.editEcsService(serviceName, capacities, "PUT")
}

func (client *SssClient) DeleteEcsService(serviceName string) (*EcsServiceResponse, error) {
	return client.getOrDeleteEcsService(serviceName, "DELETE")
}
