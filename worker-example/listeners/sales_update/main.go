package sales_update

import (
	"encoding/json"
	"fmt"
	"os"

	"worker-app/pkg/log"
	"worker-app/pkg/parameter_store"

	"worker-app/models/sales_model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func ConsumeMessages(sqsClient *sqs.SQS, queueURL string, thread int) {

	log := log.Instance()

	aws_region := os.Getenv("AWS_REGION")
	ssm_site_state_parameter := os.Getenv("SSM_PARAMETER_STORE_STATE")
	sqs_sales_queue := os.Getenv("SQS_SALES_QUEUE")

	log.Info().
		Str("Action", "consume").
		Str("Region", aws_region).
		Int("Thread", thread).
		Str("SQS_Queue", sqs_sales_queue).
		Msg("Starting Consumer Thread")

	for {

		site_state, err := parameter_store.GetParamValue(ssm_site_state_parameter, 30)

		if err != nil {
			log.Error().
				Str("Action", "consume").
				Int("Thread", thread).
				Str("Region", aws_region).
				Str("State", site_state).
				Str("SQS_Queue", sqs_sales_queue).
				Str("Error", err.Error()).
				Msg("Error to recover SSM Site State from Parameter Store")
			continue
		}

		result, err := sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: aws.Int64(10),
			WaitTimeSeconds:     aws.Int64(20),
		})

		if err != nil {
			log.Error().
				Str("Action", "consume").
				Int("Thread", thread).
				Str("Region", aws_region).
				Str("State", site_state).
				Str("SQS_Queue", sqs_sales_queue).
				Str("Error", err.Error()).
				Msg("Starting SQS Worker Service")
			continue
		}

		for _, msg := range result.Messages {

			log.Info().
				Str("Action", "consume").
				Str("Region", aws_region).
				Int("Thread", thread).
				Str("State", site_state).
				Str("SQS_Queue", sqs_sales_queue).
				Str("Body", *msg.Body).
				Msg("Message")

			// Process Message
			err := processSale(*msg.MessageId, *msg.Body, site_state, thread)

			if err == nil {
				_, err := sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
					QueueUrl:      aws.String(queueURL),
					ReceiptHandle: msg.ReceiptHandle,
				})

				if err != nil {
					log.Error().
						Str("Action", "consume").
						Str("Region", aws_region).
						Int("Thread", thread).
						Str("State", site_state).
						Str("SQS_Queue", sqs_sales_queue).
						Str("Error", err.Error()).
						Msg("Error to delete Message from Queue")
					continue
				} else {
					log.Info().
						Str("Region", aws_region).
						Str("State", site_state).
						Int("Thread", thread).
						Str("SQS_Queue", sqs_sales_queue).
						Str("MessageId", *msg.MessageId).
						Msg("Message removed from queue")
				}
			} else {
				log.Error().
					Str("Action", "consume").
					Str("Region", aws_region).
					Int("Thread", thread).
					Str("State", site_state).
					Str("SQS_Queue", sqs_sales_queue).
					Str("Error", err.Error()).
					Str("MessageId", *msg.MessageId).
					Msg("Error process sale")
				continue
			}

		}
	}
}

func processSale(id string, message string, state string, thread int) error {

	log := log.Instance()
	aws_region := os.Getenv("AWS_REGION")
	dynamodb_table := os.Getenv("DYNAMO_SALES_TABLE")

	if state != "ACTIVE" {
		log.Info().
			Str("Region", aws_region).
			Str("State", state).
			Int("Thread", thread).
			Str("MessageId", id).
			Msg("Dry-Running Message; Site is not Active")
		return nil
	}

	log.Info().
		Str("Region", aws_region).
		Str("State", state).
		Int("Thread", thread).
		Str("MessageId", id).
		Msg("Processing Message; Site is Active")

	sale := sales_model.Model{}

	err := json.Unmarshal([]byte(message), &sale)
	if err != nil {
		return err
	}

	sale.Processed = true

	err = updateOnDynamoDB(sale, dynamodb_table, state, thread)
	if err != nil {
		return err
	}

	return nil
}

func updateOnDynamoDB(sale sales_model.Model, table string, state string, thread int) error {
	log := log.Instance()
	aws_region := os.Getenv("AWS_REGION")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		return err
	}

	dbClient := dynamodb.New(sess)

	log.Info().
		Str("Region", aws_region).
		Str("State", state).
		Int("Thread", thread).
		Str("Id", sale.ID).
		Str("Product", sale.Product).
		Float64("Amount", sale.Amount).
		Bool("Processed", sale.Processed).
		Msg("Updating flag on DynamoDB Table")

	updateExpression := fmt.Sprintf("SET sale_processed = :value")
	expressionValues := map[string]*dynamodb.AttributeValue{
		":value": {
			BOOL: aws.Bool(sale.Processed),
		},
	}

	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(table),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(sale.ID),
			},
		},
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionValues,
	}

	_, err = dbClient.UpdateItem(updateInput)
	if err != nil {
		return err
	}

	log.Info().
		Str("Region", aws_region).
		Str("State", state).
		Int("Thread", thread).
		Str("Id", sale.ID).
		Str("Product", sale.Product).
		Float64("Amount", sale.Amount).
		Bool("Processed", sale.Processed).
		Msg("Sale Processed")

	return nil
}