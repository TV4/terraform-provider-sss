// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package client

import "net/http"

func (client *SssClient) GetValkeyShardScaling(serviceID string) (*ValkeyShardScalingResponse, error) {
	return getScheduledScaling[ValkeyShardScalingResponse](client, scheduledScalingServiceValkeyShards, serviceID)
}

func (client *SssClient) CreateValkeyShardScaling(serviceID string, body ValkeyShardScalingPostBody) error {
	return client.createOrUpdateScheduledScaling(scheduledScalingServiceValkeyShards, serviceID, body, http.MethodPost, http.StatusCreated)
}

func (client *SssClient) UpdateValkeyShardScaling(serviceID string, body ValkeyShardScalingPostBody) error {
	return client.createOrUpdateScheduledScaling(scheduledScalingServiceValkeyShards, serviceID, body, http.MethodPut, http.StatusOK)
}

func (client *SssClient) DeleteValkeyShardScaling(serviceID string) error {
	return client.deleteScheduledScaling(scheduledScalingServiceValkeyShards, serviceID)
}
