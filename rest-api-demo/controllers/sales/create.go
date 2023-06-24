package sales

import (
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

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error().
			Str("Action", "create").
			Str("Error", err.Error()).
			Msg("Error to Bind JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		log.Error().
			Str("Action", "create").
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
		Str("Id", response.Id).
		Str("Product", response.Product).
		Float64("Amount", response.Amount).
		Msg("Sale registered")

	c.JSON(http.StatusCreated, response)
}
