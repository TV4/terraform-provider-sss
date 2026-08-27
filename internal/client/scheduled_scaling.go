// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type scheduledScalingService string

const scheduledScalingServiceValkeyReplicas scheduledScalingService = "valkey-replicas"
const scheduledScalingServiceValkeyShards scheduledScalingService = "valkey-shards"
const scheduledScalingServiceAuroraReaders scheduledScalingService = "aurora-readers"

func getScheduledScaling[T any](client *SssClient, service scheduledScalingService, serviceID string) (*T, error) {
	response, err := client.doScheduledScaling(service, serviceID, http.MethodGet, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	var body T
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return nil, err
	}

	return &body, nil
}

func (client *SssClient) createOrUpdateScheduledScaling(service scheduledScalingService, serviceID string, body any, method string, expectedStatus int) error {
	response, err := client.doScheduledScaling(service, serviceID, method, body, expectedStatus)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	return nil
}

func (client *SssClient) deleteScheduledScaling(service scheduledScalingService, serviceID string) error {
	response, err := client.doScheduledScaling(service, serviceID, http.MethodDelete, nil, http.StatusOK)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	return nil
}

func (client *SssClient) doScheduledScaling(service scheduledScalingService, serviceID string, method string, body any, expectedStatus int) (*http.Response, error) {
	endpoint := "/api/v1/services/" + string(service) + "/" + serviceID
	requestURL := url.URL{
		Scheme:  client.protocol,
		Host:    client.host,
		Path:    endpoint,
		RawPath: "/api/v1/services/" + string(service) + "/" + url.PathEscape(serviceID),
	}

	var requestBody *bytes.Reader
	if body != nil {
		encodedBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewReader(encodedBody)
	} else {
		requestBody = bytes.NewReader(nil)
	}

	request, err := http.NewRequest(method, requestURL.String(), requestBody)
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth(client.authUsername, client.authPassword)
	request.Header.Set("Accept", "application/json, application/problem+json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != expectedStatus {
		defer func() { _ = response.Body.Close() }()
		return nil, scheduledScalingError(method, service, serviceID, response)
	}

	return response, nil
}

func scheduledScalingError(method string, service scheduledScalingService, serviceID string, response *http.Response) error {
	message := fmt.Sprintf("failed to %s service %s/%s: %s", strings.ToLower(method), service, serviceID, response.Status)
	if response.StatusCode != http.StatusUnprocessableEntity {
		return fmt.Errorf("%s", message)
	}

	var errorModel ErrorModel
	if err := json.NewDecoder(response.Body).Decode(&errorModel); err != nil {
		return fmt.Errorf("%s", message)
	}

	details := make([]string, 0, len(errorModel.Errors)+1)
	if errorModel.Detail != "" {
		details = append(details, errorModel.Detail)
	}
	for _, detail := range errorModel.Errors {
		if detail.Location != "" && detail.Message != "" {
			details = append(details, detail.Location+": "+detail.Message)
		}
	}
	if len(details) == 0 {
		return fmt.Errorf("%s", message)
	}

	return fmt.Errorf("%s: %s", message, strings.Join(details, "; "))
}
