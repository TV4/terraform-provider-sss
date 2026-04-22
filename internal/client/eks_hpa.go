// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package client

func (client *SssClient) GetEksHpa(serviceId string) (*EksHpaResponse, error) {
	return getOrDeleteScalable[EksHpaResponse](client, scalableTypeEKSHPA, serviceId, "GET")
}

func (client *SssClient) CreateEksHpa(serviceId string, body EksHpaPostBody) error {
	return editScalable(client, scalableTypeEKSHPA, serviceId, body, "POST")
}

func (client *SssClient) UpdateEksHpa(serviceId string, body EksHpaPostBody) error {
	return editScalable(client, scalableTypeEKSHPA, serviceId, body, "PUT")
}

func (client *SssClient) DeleteEksHpa(serviceId string) (*EksHpaResponse, error) {
	return getOrDeleteScalable[EksHpaResponse](client, scalableTypeEKSHPA, serviceId, "DELETE")
}
