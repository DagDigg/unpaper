package stripeservice

import (
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/stripe/stripe-go/v72"
)

// CreateRoomSubscriptionParams parameters
// for creating a stripe subscription
type CreateRoomSubscriptionParams struct {
	SenderConnectedCustomerID string
	SenderUserID              string
	ReceiverAccountID         string
	PlatformFeePct            float64
	Amount                    int64
	ProductID                 string
	Metadata                  map[string]string
}

// StripeSubscriptionOptions are options
// used when converting the subscription to protobuf
type StripeSubscriptionOptions struct {
	WithLatestInvoice bool
	WithItems         bool
	WithPlan          bool
}

// StorePlatformSubscriptionParams parameters
// for stroring a stripe subscription on db
type StorePlatformSubscriptionParams struct {
	Subscription *stripe.Subscription
}

// StoreRoomSubscriptionParams parameters for
// storing a room subscription on db
type StoreRoomSubscriptionParams struct {
	Subscription *stripe.Subscription
	// Optional invoice that will override 'Subscription.LatestInvoice'
	Invoice   *stripe.Invoice
	RoomID    string
	Options   *StripeSubscriptionOptions
	AccountID string
}

// StripeSubscriptionToPBParams parameters
// for converting a stripe subscription to protobuf
type StripeSubscriptionToPBParams struct {
	Subscription *stripe.Subscription
	Options      *StripeSubscriptionOptions
}

// UpdatePlatformSubscriptionParams parameters
// for updating a stripe platform subscription
type UpdatePlatformSubscriptionParams struct {
	Plan     v1API.Plan
	Customer *stripe.Customer
}

// GetPlatformInvoicePreviewParams parameters for getting
// a stripe invoice preview
type GetPlatformInvoicePreviewParams struct {
	CustomerID         string
	SubscriptionID     string
	SubscriptionItemID string
	Plan               v1API.Plan
	Coupon             string
}

// GetRoomSubscriptionForUserIDOptions options used
// when getting user subscription for room
type GetRoomSubscriptionForUserIDOptions struct {
	StrictPaymentIntentCheck bool
}

// GetRoomSubscriptionForUserIDParams params for get
type GetRoomSubscriptionForUserIDParams struct {
	RoomID    string
	UserID    string
	AccountID string
	Options   GetRoomSubscriptionForUserIDOptions
}
