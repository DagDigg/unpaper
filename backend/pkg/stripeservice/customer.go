package stripeservice

// CreateCustomerParams params for creating a stripe customer
type CreateCustomerParams struct {
	UserID     string
	Email      string
	FamilyName string
	GivenName  string
}

// StoreCustomerParams params for storing a customer
type StoreCustomerParams struct {
	UserID     string
	FamilyName string
	GivenName  string
}

// CreateConnectedCustomerParams params for
// creating a connected customer on an account id
type CreateConnectedCustomerParams struct {
	CustomerID string
	AccountID  string
}

// GetOrCreateConnectedCustomerParams params for
// retrieving or creating a connected customer on an account id
type GetOrCreateConnectedCustomerParams struct {
	UserID     string
	CustomerID string
	AccountID  string
}
