package sales_model

type Model struct {
	ID        string  `dynamodbav:"id"`
	Product   string  `dynamodbav:"product"`
	Amount    float64 `dynamodbav:"amount"`
	Timestamp int64   `dynamodbav:"timestamp"`
	// Outros campos
}
