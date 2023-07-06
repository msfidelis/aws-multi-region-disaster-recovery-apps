package sales_model

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type ModelDAO struct {
	tableName string
	client    *dynamodb.DynamoDB
}

func NewModelDAO(client *dynamodb.DynamoDB) *ModelDAO {
	table_name := os.Getenv("DYNAMO_SALES_TABLE")
	return &ModelDAO{
		tableName: table_name,
		client:    client,
	}
}

func (dao *ModelDAO) Create(model *Model) error {
	av, err := dynamodbattribute.MarshalMap(model)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(dao.tableName),
	}

	_, err = dao.client.PutItem(input)
	return err
}

func (dao *ModelDAO) GetByID(id string) (*Model, error) {
	input := &dynamodb.QueryInput{
		TableName:              &dao.tableName,
		KeyConditionExpression: aws.String("id = :value"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":value": {
				S: aws.String(id),
			},
		},
	}

	result, err := dao.client.Query(input)
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	model := &Model{}
	err = dynamodbattribute.UnmarshalMap(result.Items[0], model)
	if err != nil {
		return nil, err
	}

	return model, nil
}

func (dao *ModelDAO) Delete(id string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(dao.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
	}

	_, err := dao.client.DeleteItem(input)
	if err != nil {
		return err
	}

	return nil
}

func (dao *ModelDAO) UpdatedProcessedFlag(id string) error {
	key := map[string]*dynamodb.AttributeValue{
		"id": {
			S: aws.String(id),
		},
	}

	updateExpression := "SET " + "sale_processed" + " = :value"
	expressionAttributeValues := map[string]*dynamodb.AttributeValue{
		":value": {
			BOOL: aws.Bool(true),
		},
	}

	input := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(dao.tableName),
		Key:                       key,
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionAttributeValues,
	}

	_, err := dao.client.UpdateItem(input)
	return err
}
