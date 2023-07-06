package sales

import (
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gin-gonic/gin"
	"github.com/msfidelis/sales-rest-api/models/sales_model"
	"github.com/msfidelis/sales-rest-api/pkg/log"
	"github.com/msfidelis/sales-rest-api/pkg/parameter_store"
)

// Sales godoc
// @Summary Get a single item from Sales catalog
// @Tags Sales
// @Produce json
// @Success 200 {object} Response
// @Router /sales/:id [get]
func GetByID(c *gin.Context) {

	var response Response

	log := log.Instance()

	ssm_site_state_parameter := os.Getenv("SSM_PARAMETER_STORE_STATE")
	aws_region := os.Getenv("AWS_REGION")

	site_state, err := parameter_store.GetParamValue(ssm_site_state_parameter, 30)

	if err != nil {
		log.Error().
			Str("Action", "read").
			Str("Region", aws_region).
			Str("State", site_state).
			Str("Error", err.Error()).
			Msg("Error to recover site state from parameter store")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv(aws_region)),
	})
	if err != nil {
		log.Error().
			Str("Action", "read").
			Str("Region", aws_region).
			Str("State", site_state).
			Str("Error", err.Error()).
			Msg("Error to read DynamoDB Session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	svc := dynamodb.New(sess)
	dao := sales_model.NewModelDAO(svc)

	sale, err := dao.GetByID(id)
	if err != nil {
		log.Error().
			Str("Action", "read").
			Str("Region", aws_region).
			Str("State", site_state).
			Str("Error", err.Error()).
			Msg("Error to execute DynamoDB Query")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if sale == nil {
		log.Warn().
			Str("Action", "read").
			Str("Region", aws_region).
			Str("State", site_state).
			Str("Id", id).
			Str("Error", err.Error()).
			Msg("Item not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	response.Id = sale.ID
	response.Amount = sale.Amount
	response.Product = sale.Product

	c.JSON(http.StatusOK, response)

}
