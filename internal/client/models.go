// Copyright (c) TV4 Media AB
// SPDX-License-Identifier: MPL-2.0

package client

type EcsServicePostBody struct {
	MinExtremeCapacity int64 `json:"minExtremeCapacity"`
	MinHighCapacity    int64 `json:"minHighCapacity"`
	MinMediumCapacity  int64 `json:"minMediumCapacity"`
	MinLowCapacity     int64 `json:"minLowCapacity"`
}

type EcsServiceResponse struct {
	Name               string `json:"name"`
	MinExtremeCapacity int64  `json:"minExtremeCapacity"`
	MinHighCapacity    int64  `json:"minHighCapacity"`
	MinMediumCapacity  int64  `json:"minMediumCapacity"`
	MinLowCapacity     int64  `json:"minLowCapacity"`
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
