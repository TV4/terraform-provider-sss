// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package client

func (client *SssClient) GetEcsService(serviceName string) (*EcsServiceResponse, error) {
	return getOrDeleteScalable[EcsServiceResponse](client, scalableTypeECS, serviceName, "GET")
}

func (client *SssClient) CreateEcsService(serviceName string, capacities EcsServicePostBody) error {
	return editScalable(client, scalableTypeECS, serviceName, capacities, "POST")
}

func (client *SssClient) UpdateEcsService(serviceName string, capacities EcsServicePostBody) error {
	return editScalable(client, scalableTypeECS, serviceName, capacities, "PUT")
}

func (client *SssClient) DeleteEcsService(serviceName string) (*EcsServiceResponse, error) {
	return getOrDeleteScalable[EcsServiceResponse](client, scalableTypeECS, serviceName, "DELETE")
}
