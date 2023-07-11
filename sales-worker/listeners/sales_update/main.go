package sales_update

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"sales-worker/models/sales_model"
	"sales-worker/pkg/log"
	"sales-worker/pkg/parameter_store"
	"sales-worker/pkg/s3"

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

	if state != "ACTIVE" {
		log.Info().
			Str("Region", aws_region).
			Str("State", state).
			Int("Thread", thread).
			Str("MessageId", id).
			Msg("Dry-Running Message; Site is not Active")
		return nil
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(aws_region),
	})
	if err != nil {
		return err
	}

	svc := dynamodb.New(sess)
	dao := sales_model.NewModelDAO(svc)

	log.Info().
		Str("Region", aws_region).
		Str("State", state).
		Int("Thread", thread).
		Str("Sale", id).
		Msg("Processing Message; Site is Active")

	sale := sales_model.Model{}

	err = json.Unmarshal([]byte(message), &sale)
	if err != nil {
		return err
	}

	log.Info().
		Str("Region", aws_region).
		Str("State", state).
		Int("Thread", thread).
		Str("Sale", sale.ID).
		Msg("Checking Idempotency")

	idempotency, err := dao.CheckIdempotency(sale.ID)
	if err != nil {
		return err
	}

	if idempotency {
		log.Info().
			Str("Region", aws_region).
			Str("State", state).
			Int("Thread", thread).
			Str("Sale", sale.ID).
			Msg("Sale already processed, item found in idempotency table")
		return nil
	}

	sale.Processed = true
	err = updateOnDynamoDB(sale, state, thread)
	if err != nil {
		return err
	}

	err = saveSaleOnS3(sale.ID, message, state, thread)
	if err != nil {
		return err
	}

	err = dao.SetIdempotency(sale.ID)
	if err != nil {
		return err
	}

	log.Info().
		Str("Region", aws_region).
		Str("State", state).
		Int("Thread", thread).
		Str("MessageId", id).
		Msg("Sale saved on idempotency table")

	return nil
}

func updateOnDynamoDB(pre_sale sales_model.Model, state string, thread int) error {
	log := log.Instance()
	aws_region := os.Getenv("AWS_REGION")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		return err
	}

	svc := dynamodb.New(sess)
	dao := sales_model.NewModelDAO(svc)

	sale, err := dao.GetByID(pre_sale.ID)
	if err != nil {
		log.Error().
			Str("Region", aws_region).
			Str("State", state).
			Str("Error", err.Error()).
			Msg("Error to execute DynamoDB Query")
		return err
	}
	if sale == nil {
		log.Warn().
			Str("Action", "read").
			Str("Region", aws_region).
			Str("State", state).
			Str("Id", pre_sale.ID).
			Msg("Sale not found")
		return errors.New("sale not found")
	}

	log.Info().
		Str("Region", aws_region).
		Str("State", state).
		Int("Thread", thread).
		Str("Id", sale.ID).
		Str("Product", sale.Product).
		Float64("Amount", sale.Amount).
		Bool("Processed", sale.Processed).
		Msg("Updating flag on DynamoDB Table")

	err = dao.UpdatedProcessedFlag(sale.ID)
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
		Msg("Sale flag updated")

	return nil
}

func saveSaleOnS3(id string, message string, state string, thread int) error {
	log := log.Instance()

	now := time.Now()
	formato := "20060102"
	string_now := now.Format(formato)

	aws_region := os.Getenv("AWS_REGION")
	bucket := os.Getenv("S3_SALES_BUCKET")
	key := fmt.Sprintf("sales/%s/%s.json", string_now, id)
	log.Info().
		Str("Region", aws_region).
		Str("State", state).
		Int("Thread", thread).
		Str("Bucket", bucket).
		Str("Sale", id).
		Msg("Uploading Sale to S3")

	bytes := []byte(message)
	err := s3.Save(bytes, bucket, key)
	if err != nil {
		log.Error().
			Str("Region", aws_region).
			Str("State", state).
			Int("Thread", thread).
			Str("Bucket", bucket).
			Str("Sale", id).
			Str("Path", key).
			Str("Error", err.Error()).
			Msg("Error to save sale on S3")
		return err
	}

	log.Info().
		Str("Region", aws_region).
		Str("State", state).
		Int("Thread", thread).
		Str("Bucket", bucket).
		Str("Sale", id).
		Str("Path", key).
		Msg("Sale saved on S3")

	return nil
}

// func checkIdempotency(id string, state string, thread int) bool {

// }

// func saveIdempotency(id string, state string, thread int) error {
// log := log.Instance()
// aws_region := os.Getenv("AWS_REGION")
// idempotency_table := os.Getenv("DYNAMO_SALES_IDEMPOTENCY_TABLE")

// sess := session.Must(session.NewSessionWithOptions(session.Options{
// 	SharedConfigState: session.SharedConfigEnable,
// }))
// }
