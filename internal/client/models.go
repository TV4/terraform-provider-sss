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
	TableName       string              `json:"tableName"`
	Region          string              `json:"region"`
	LowCapacity     DynamoTableCapacity `json:"lowCapacity"`
	MediumCapacity  DynamoTableCapacity `json:"mediumCapacity"`
	HighCapacity    DynamoTableCapacity `json:"highCapacity"`
	ExtremeCapacity DynamoTableCapacity `json:"extremeCapacity"`
}

type EksHpaPostBody struct {
	Cluster    string `json:"cluster"`
	Region     string `json:"region"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	MinLow     int64  `json:"minLow"`
	MinMedium  int64  `json:"minMedium"`
	MinHigh    int64  `json:"minHigh"`
	MinExtreme int64  `json:"minExtreme"`
}

type EksHpaResponse struct {
	ID         string `json:"id"`
	Cluster    string `json:"cluster"`
	Region     string `json:"region"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	MinLow     int64  `json:"minLow"`
	MinMedium  int64  `json:"minMedium"`
	MinHigh    int64  `json:"minHigh"`
	MinExtreme int64  `json:"minExtreme"`
}

type ValkeyReplicaScalingPostBody struct {
	Region                 string `json:"region"`
	ScaleUpLeadTimeMinutes int64  `json:"scaleUpLeadTimeMinutes"`
	ReplicaCountLow        int64  `json:"replicaCountLow"`
	ReplicaCountMedium     int64  `json:"replicaCountMedium"`
	ReplicaCountHigh       int64  `json:"replicaCountHigh"`
	ReplicaCountExtreme    int64  `json:"replicaCountExtreme"`
}

type ValkeyReplicaScalingResponse struct {
	ServiceID              string `json:"serviceId"`
	Region                 string `json:"region"`
	ScaleUpLeadTimeMinutes int64  `json:"scaleUpLeadTimeMinutes"`
	ReplicaCountLow        int64  `json:"replicaCountLow"`
	ReplicaCountMedium     int64  `json:"replicaCountMedium"`
	ReplicaCountHigh       int64  `json:"replicaCountHigh"`
	ReplicaCountExtreme    int64  `json:"replicaCountExtreme"`
}

type ValkeyShardCapacity struct {
	MinShardCount int64 `json:"minShardCount"`
	MaxShardCount int64 `json:"maxShardCount"`
}

type ValkeyShardScalingPostBody struct {
	Region                 string              `json:"region"`
	ScaleUpLeadTimeMinutes int64               `json:"scaleUpLeadTimeMinutes"`
	LowCapacity            ValkeyShardCapacity `json:"lowCapacity"`
	MediumCapacity         ValkeyShardCapacity `json:"mediumCapacity"`
	HighCapacity           ValkeyShardCapacity `json:"highCapacity"`
	ExtremeCapacity        ValkeyShardCapacity `json:"extremeCapacity"`
}

type ValkeyShardScalingResponse struct {
	ServiceID              string              `json:"serviceId"`
	Region                 string              `json:"region"`
	ScaleUpLeadTimeMinutes int64               `json:"scaleUpLeadTimeMinutes"`
	LowCapacity            ValkeyShardCapacity `json:"lowCapacity"`
	MediumCapacity         ValkeyShardCapacity `json:"mediumCapacity"`
	HighCapacity           ValkeyShardCapacity `json:"highCapacity"`
	ExtremeCapacity        ValkeyShardCapacity `json:"extremeCapacity"`
}

type AuroraReaderCapacity struct {
	MinReaders int64 `json:"minReaders"`
	MaxReaders int64 `json:"maxReaders"`
}

type AuroraReaderScalingPostBody struct {
	Region                 string               `json:"region"`
	ScaleUpLeadTimeMinutes int64                `json:"scaleUpLeadTimeMinutes"`
	LowCapacity            AuroraReaderCapacity `json:"lowCapacity"`
	MediumCapacity         AuroraReaderCapacity `json:"mediumCapacity"`
	HighCapacity           AuroraReaderCapacity `json:"highCapacity"`
	ExtremeCapacity        AuroraReaderCapacity `json:"extremeCapacity"`
}

type AuroraReaderScalingResponse struct {
	ServiceID              string               `json:"serviceId"`
	Region                 string               `json:"region"`
	ScaleUpLeadTimeMinutes int64                `json:"scaleUpLeadTimeMinutes"`
	LowCapacity            AuroraReaderCapacity `json:"lowCapacity"`
	MediumCapacity         AuroraReaderCapacity `json:"mediumCapacity"`
	HighCapacity           AuroraReaderCapacity `json:"highCapacity"`
	ExtremeCapacity        AuroraReaderCapacity `json:"extremeCapacity"`
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
