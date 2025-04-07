package client

func (client *SssClient) GetDynamoTable(tableArn string) (*DynamoTableResponse, error) {
	return getOrDeleteScalable[DynamoTableResponse](client, scalableTypeDynamoDB, tableArn, "GET")
}

func (client *SssClient) CreateDynamoTable(tableArn string, capacities DynamoTablePostBody) error {
	return editScalable(client, scalableTypeDynamoDB, tableArn, capacities, "POST")
}

func (client *SssClient) UpdateDynamoTable(tableArn string, capacities DynamoTablePostBody) error {
	return editScalable(client, scalableTypeDynamoDB, tableArn, capacities, "PUT")
}

func (client *SssClient) DeleteDynamoTable(tableArn string) (*DynamoTableResponse, error) {
	return getOrDeleteScalable[DynamoTableResponse](client, scalableTypeDynamoDB, tableArn, "DELETE")
}
