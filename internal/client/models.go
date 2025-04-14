// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package client

type EcsServicePostBody struct {
	MinExtremeCapacity int64  `json:"minExtremeCapacity"`
	MinHighCapacity    int64  `json:"minHighCapacity"`
	MinMediumCapacity  int64  `json:"minMediumCapacity"`
	MinLowCapacity     int64  `json:"minLowCapacity"`
	Region             string `json:"region"`
}

type EcsServiceResponse struct {
	Name               string `json:"name"`
	MinExtremeCapacity int64  `json:"minExtremeCapacity"`
	MinHighCapacity    int64  `json:"minHighCapacity"`
	MinMediumCapacity  int64  `json:"minMediumCapacity"`
	MinLowCapacity     int64  `json:"minLowCapacity"`
	Region             string `json:"region"`
}

type DynamoTableCapacity struct {
	MinWriteCapacity int64 `json:"minWriteCapacity"`
	MinReadCapacity  int64 `json:"minReadCapacity"`
	MaxWriteCapacity int64 `json:"maxWriteCapacity"`
	MaxReadCapacity  int64 `json:"maxReadCapacity"`
}

type DynamoTablePostBody struct {
	Region          string              `json:"region"`
	LowCapacity     DynamoTableCapacity `json:"lowCapacity"`
	MediumCapacity  DynamoTableCapacity `json:"mediumCapacity"`
	HighCapacity    DynamoTableCapacity `json:"highCapacity"`
	ExtremeCapacity DynamoTableCapacity `json:"extremeCapacity"`
}

type DynamoTableResponse struct {
	TableArn        string              `json:"tableArn"`
	Region          string              `json:"region"`
	LowCapacity     DynamoTableCapacity `json:"lowCapacity"`
	MediumCapacity  DynamoTableCapacity `json:"mediumCapacity"`
	HighCapacity    DynamoTableCapacity `json:"highCapacity"`
	ExtremeCapacity DynamoTableCapacity `json:"extremeCapacity"`
}

type ErrorDetail struct {
	Location string `json:"location"`
	Message  string `json:"message"`
	Value    any    `json:"value"`
}

type ErrorModel struct {
	Detail   string        `json:"detail"`
	Errors   []ErrorDetail `json:"errors"`
	Instance string        `json:"instance"`
	Status   int           `json:"status"`
	Title    string        `json:"title"`
	Type     string        `json:"type"`
}
