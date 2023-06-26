package sales

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gin-gonic/gin"
	guuid "github.com/google/uuid"
	"github.com/msfidelis/rest-api-demo/models/sales_model"
	"github.com/msfidelis/rest-api-demo/pkg/log"
	"github.com/msfidelis/rest-api-demo/pkg/parameter_store"
	"github.com/msfidelis/rest-api-demo/pkg/sns"
)

type Request struct {
	Product string  `json:"product" binding:"required"`
	Amount  float64 `json:"amount" binding:"required"`
}

type Response struct {
	Id        string  `json:"id" binding:"id"`
	Product   string  `json:"product" binding:"required"`
	Amount    float64 `json:"amount" binding:"required"`
	Processed bool    `json:"processed" binding:"required"`
}

// Sales godoc
// @Summary Create a Sale Item on DynamoDB
// @Tags Sales
// @Produce json
// @Success 200 {object} Response
// @Router /sales [post]
func Create(c *gin.Context) {
	var request Request
	var response Response

	log := log.Instance()

	sns_processing_topic := os.Getenv("SNS_SALES_PROCESSING_TOPIC")
	ssm_site_state_parameter := os.Getenv("SSM_PARAMETER_STORE_STATE")
	aws_region := os.Getenv("AWS_REGION")

	site_state, err := parameter_store.GetParamValue(ssm_site_state_parameter, 30)

	if err != nil {
		log.Error().
			Str("Action", "create").
			Str("Region", aws_region).
			Str("State", site_state).
			Str("Error", err.Error()).
			Msg("Error to recover site state from parameter store")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error().
			Str("Action", "create").
			Str("Region", aws_region).
			Str("State", site_state).
			Str("Error", err.Error()).
			Msg("Error to Bind JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(aws_region),
	})
	if err != nil {
		log.Error().
			Str("Action", "create").
			Str("Region", aws_region).
			Str("State", site_state).
			Str("Error", err.Error()).
			Msg("Error to create DynamoDB Session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	svc := dynamodb.New(sess)

	saleModel := &sales_model.Model{
		ID:        guuid.New().String(),
		Product:   request.Product,
		Amount:    request.Amount,
		Timestamp: time.Now().Unix(),
		Processed: false,
	}

	dao := sales_model.NewModelDAO(svc)

	err = dao.Create(saleModel)
	if err != nil {
		log.Error().
			Str("Action", "create").
			Str("Region", aws_region).
			Str("State", site_state).
			Str("Error", err.Error()).
			Msg("Error to save item to dynamoDB")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response.Id = saleModel.ID
	response.Product = saleModel.Product
	response.Amount = saleModel.Amount
	response.Processed = saleModel.Processed

	log.Info().
		Str("Action", "create").
		Str("Region", aws_region).
		Str("State", site_state).
		Str("Id", response.Id).
		Str("Product", response.Product).
		Float64("Amount", response.Amount).
		Msg("Sale persisted on DynamoDB")

	json_string, err := json.Marshal(saleModel)
	if err != nil {
		log.Error().
			Str("Action", "create").
			Str("Region", aws_region).
			Str("State", site_state).
			Str("Error", err.Error()).
			Msg("Error to marshall model on JSON String")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = sns.Publish(string(json_string), sns_processing_topic)

	if err != nil {
		log.Error().
			Str("Action", "create").
			Str("Region", aws_region).
			Str("State", site_state).
			Str("Error", err.Error()).
			Str("SNS_Topic", sns_processing_topic).
			Msg("Failed to publish message on SALES_PROCESSING topic")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Str("Action", "create").
		Str("Region", aws_region).
		Str("State", site_state).
		Str("SNS_Topic", sns_processing_topic).
		Str("Id", response.Id).
		Str("Product", response.Product).
		Float64("Amount", response.Amount).
		Msg("Sale processing event published on SNS Topic")

	c.JSON(http.StatusCreated, response)
}
