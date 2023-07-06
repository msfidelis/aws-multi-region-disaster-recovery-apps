package main

import (
	"os"
	"os/signal"
	"sales-worker/pkg/log"
	"strconv"
	"syscall"

	"sales-worker/pkg/parameter_store"

	"sales-worker/listeners/sales_update"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func main() {

	log := log.Instance()

	aws_region := os.Getenv("AWS_REGION")
	ssm_site_state_parameter := os.Getenv("SSM_PARAMETER_STORE_STATE")
	sqs_sales_queue := os.Getenv("SQS_SALES_QUEUE")
	threads := os.Getenv("CONSUMER_THREADS")

	num_threads, err := strconv.Atoi(threads)
	if err != nil {
		log.Error().
			Str("Action", "consume").
			Str("Region", aws_region).
			Str("Threads", threads).
			Str("SQS_Queue", sqs_sales_queue).
			Str("Error", err.Error()).
			Msg("Erro to convert CONSUMER_THREAD to int64; Assuming default value: 2")
		num_threads = 2
	}

	if num_threads <= 0 {
		num_threads = 2
	}

	site_state, err := parameter_store.GetParamValue(ssm_site_state_parameter, 30)

	if err != nil {
		log.Error().
			Str("Action", "consume").
			Str("Region", aws_region).
			Str("State", site_state).
			Str("SQS_Queue", sqs_sales_queue).
			Str("Error", err.Error()).
			Msg("Error to recover SSM Site State from Parameter Store")
		os.Exit(1)
	}

	log.Info().
		Str("SQS_Queue", sqs_sales_queue).
		Msg("Starting SQS Worker Service")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(aws_region),
	})
	if err != nil {
		log.Error().
			Str("Action", "consume").
			Str("Region", aws_region).
			Str("State", site_state).
			Str("SQS_Queue", sqs_sales_queue).
			Str("Error", err.Error()).
			Msg("Error to create SQS Session")
		return
	}

	// Iniciar o consumo de mensagens da fila SQS
	for i := 0; i < num_threads; i++ {
		sqsClient := sqs.New(sess)
		go sales_update.ConsumeMessages(sqsClient, sqs_sales_queue, i)
	}

	waitForExitSignal()
}

func waitForExitSignal() {
	log := log.Instance()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	log.Info().Msg("Stopping consumer")
	os.Exit(0)
}
