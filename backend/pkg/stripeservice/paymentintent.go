package stripeservice

// CreateConnectPaymentIntentParams parameters used for creating a payment intent
type CreateConnectPaymentIntentParams struct {
	PlatformFeePercent      float64
	SenderConnectCustomerID string
	ReceiverAccountID       string
	Amount                  int64
	Metadata                map[string]string
}

// ConfirmConnectPaymentIntentParams parameters for confirming a
// stripe connect payment intent.
type ConfirmConnectPaymentIntentParams struct {
	// PaymentIntentID id of the payment intent we want to confirm
	PaymentIntentID string
	// AccountID of the owner of the payment intent
	AccountID string
	// CustomerID of connected customer id
	CustomerID string
}
