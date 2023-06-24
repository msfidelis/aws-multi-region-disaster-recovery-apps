package sns

import (
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

func Publish(message string, topic_arn string) (*sns.PublishOutput, error) {

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")), // Substitua pela região desejada
	})

	if err != nil {
		log.Fatal("Erro ao criar a sessão da AWS:", err)
	}

	svc := sns.New(sess)

	result, err := svc.Publish(&sns.PublishInput{
		Message:  aws.String(message),
		TopicArn: aws.String(topic_arn),
	})

	return result, err
}
