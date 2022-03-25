package stripeservice

// SyncConnectPaymentMethodsParams parameters for
// synching payment methods between platform and connect account
type SyncConnectPaymentMethodsParams struct {
	CustomerID          string
	ConnectedCustomerID string
	AccountID           string
}
