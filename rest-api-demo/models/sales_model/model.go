package sales_model

type Model struct {
	ID        string  `dynamodbav:"id" json:"id"`
	Product   string  `dynamodbav:"product" json:"product"`
	Amount    float64 `dynamodbav:"amount" json:"amount"`
	Processed bool    `dynamodbav:"sale_processed" json:"sale_processed"`
	Timestamp int64   `dynamodbav:"timestamp" json:"timestamp"`
}
