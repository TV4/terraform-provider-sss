// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package client

import "net/http"

func (client *SssClient) GetAuroraReaderScaling(serviceID string) (*AuroraReaderScalingResponse, error) {
	return getScheduledScaling[AuroraReaderScalingResponse](client, scheduledScalingServiceAuroraReaders, serviceID)
}

func (client *SssClient) CreateAuroraReaderScaling(serviceID string, body AuroraReaderScalingPostBody) error {
	return client.createOrUpdateScheduledScaling(scheduledScalingServiceAuroraReaders, serviceID, body, http.MethodPost, http.StatusCreated)
}

func (client *SssClient) UpdateAuroraReaderScaling(serviceID string, body AuroraReaderScalingPostBody) error {
	return client.createOrUpdateScheduledScaling(scheduledScalingServiceAuroraReaders, serviceID, body, http.MethodPut, http.StatusOK)
}

func (client *SssClient) DeleteAuroraReaderScaling(serviceID string) error {
	return client.deleteScheduledScaling(scheduledScalingServiceAuroraReaders, serviceID)
}
