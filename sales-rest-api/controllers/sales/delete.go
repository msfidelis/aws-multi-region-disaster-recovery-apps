package sales

import (
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gin-gonic/gin"
	"github.com/msfidelis/sales-rest-api/models/sales_model"
)

// Sales godoc
// @Summary Delete a single item from Sales catalog
// @Tags Sales
// @Produce json
// @Success 200 {object} Response
// @Router /sales/:id [delete]
func DeleteById(c *gin.Context) {

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

	err = dao.Delete(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}
