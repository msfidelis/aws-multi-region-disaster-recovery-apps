package sales

import (
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gin-gonic/gin"
	"github.com/msfidelis/rest-api-demo/models/sales_model"
)

// Sales godoc
// @Summary Get a single item from Sales catalog
// @Tags Sales
// @Produce json
// @Success 200 {object} Response
// @Router /sales/:id [get]
func GetByID(c *gin.Context) {

	var response Response

	id := c.Param("id")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	svc := dynamodb.New(sess)
	dao := sales_model.NewModelDAO(svc)

	sale, err := dao.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if sale == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	response.Id = sale.ID
	response.Amount = sale.Amount
	response.Product = sale.Product

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)

}
