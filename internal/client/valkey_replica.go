// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package client

import "net/http"

func (client *SssClient) GetValkeyReplicaScaling(serviceID string) (*ValkeyReplicaScalingResponse, error) {
	return getScheduledScaling[ValkeyReplicaScalingResponse](client, scheduledScalingServiceValkeyReplicas, serviceID)
}

func (client *SssClient) CreateValkeyReplicaScaling(serviceID string, body ValkeyReplicaScalingPostBody) error {
	return client.createOrUpdateScheduledScaling(scheduledScalingServiceValkeyReplicas, serviceID, body, http.MethodPost, http.StatusCreated)
}

func (client *SssClient) UpdateValkeyReplicaScaling(serviceID string, body ValkeyReplicaScalingPostBody) error {
	return client.createOrUpdateScheduledScaling(scheduledScalingServiceValkeyReplicas, serviceID, body, http.MethodPut, http.StatusOK)
}

func (client *SssClient) DeleteValkeyReplicaScaling(serviceID string) error {
	return client.deleteScheduledScaling(scheduledScalingServiceValkeyReplicas, serviceID)
}
