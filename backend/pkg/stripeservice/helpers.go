package stripeservice

import (
	"fmt"
	"math"

	"github.com/DagDigg/unpaper/backend/customers"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/core/config"
	"github.com/stripe/stripe-go/v72"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// calcFeeAmount calculates price fees by adding the platform
// fee percentage and stripe fees. It returns an integer amount
// that express the cents for the fee
func calcFeeAmount(total float64, feePct float64) int {
	return int((total * feePct) / 100)
}

// calcFeePercent calculates price fees by adding the platform
// fee percentage and stripe fees. It returns a percentage that
// should be calculated against the total price
func calcFeePercent(total float64, feePct float64) float64 {
	platformAmt := total * feePct / 100
	stripeAmt := calcStripeFee(total)
	fullAmt := platformAmt + stripeAmt
	res := fullAmt * 100 / total
	// Return a 2 decimal places number
	return math.Floor(res*100) / 100
}

func calcStripeFee(total float64) float64 {
	return (total * 2.9 / 100) + 0.3
}

// stripeSubscriptionStatusToPB converts stripe subscription status to protobuf enum
func stripeSubscriptionStatusToPB(status stripe.SubscriptionStatus) (v1API.SubscriptionStatus, error) {
	switch status {
	case stripe.SubscriptionStatusActive:
		return v1API.SubscriptionStatus_ACTIVE, nil
	case stripe.SubscriptionStatusCanceled:
		return v1API.SubscriptionStatus_CANCELED, nil
	case stripe.SubscriptionStatusIncomplete:
		return v1API.SubscriptionStatus_INCOMPLETE, nil
	case stripe.SubscriptionStatusIncompleteExpired:
		return v1API.SubscriptionStatus_INCOMPLETE_EXPIRED, nil
	case stripe.SubscriptionStatusPastDue:
		return v1API.SubscriptionStatus_PAST_DUE, nil
	case stripe.SubscriptionStatusTrialing:
		return v1API.SubscriptionStatus_TRIALING, nil
	case stripe.SubscriptionStatusUnpaid:
		return v1API.SubscriptionStatus_UNPAID, nil
	default:
		return 0, fmt.Errorf("failed to convert stripe subscription status: %v", status)
	}
}

// StripePaymentIntentToPB converts a stripe payment intent to protobuf
func StripePaymentIntentToPB(pi *stripe.PaymentIntent) (*v1API.PaymentIntent, error) {
	piStatus, err := StripePaymentIntentStatusToPB(pi.Status)
	if err != nil {
		return nil, err
	}

	return &v1API.PaymentIntent{
		Status:       piStatus,
		ClientSecret: pi.ClientSecret,
		Id:           pi.ID,
	}, nil
}

// StripePaymentIntentStatusToPB converts stripe payment intent status to protobuf enum
func StripePaymentIntentStatusToPB(status stripe.PaymentIntentStatus) (v1API.PaymentIntentStatus_Enum, error) {
	switch status {
	case stripe.PaymentIntentStatusCanceled:
		return v1API.PaymentIntentStatus_CANCELED, nil
	case stripe.PaymentIntentStatusProcessing:
		return v1API.PaymentIntentStatus_PROCESSING, nil
	case stripe.PaymentIntentStatusRequiresAction:
		return v1API.PaymentIntentStatus_REQUIRES_ACTION, nil
	case stripe.PaymentIntentStatusRequiresCapture:
		return v1API.PaymentIntentStatus_REQUIRES_CAPTURE, nil
	case stripe.PaymentIntentStatusRequiresConfirmation:
		return v1API.PaymentIntentStatus_REQUIRES_CONFIRMATION, nil
	case stripe.PaymentIntentStatusRequiresPaymentMethod:
		return v1API.PaymentIntentStatus_REQUIRES_PAYMENT_METHOD, nil
	case stripe.PaymentIntentStatusSucceeded:
		return v1API.PaymentIntentStatus_SUCCEEDED, nil
	default:
		return 0, fmt.Errorf("invalid stripe payment intent status: %v ", status)
	}
}

// pbPlanToPriceID converts protobuf plan enum to config priceID (the one provided by stripe)
func pbPlanToPriceID(cfg *config.Config, plan v1API.Plan) (string, error) {
	switch plan {
	case v1API.Plan_UNPAPER_FREE:
		return cfg.PriceIDFree, nil
	case v1API.Plan_UNPAPER_PLUS_MONTHLY:
		return cfg.PriceIDPlusMonthly, nil
	case v1API.Plan_UNPAPER_PLUS_YEARLY:
		return cfg.PriceIDPlusYearly, nil
	default:
		return "", fmt.Errorf("failed to convert PB plan to priceID: %q", plan)
	}
}

// stripePriceIDToPBPlan converts stripe price id to protobuf plan enum
func stripePriceIDToPBPlan(cfg *config.Config, priceID string) (v1API.Plan, error) {
	switch priceID {
	case cfg.PriceIDFree:
		return v1API.Plan_UNPAPER_FREE, nil
	case cfg.PriceIDPlusMonthly:
		return v1API.Plan_UNPAPER_PLUS_MONTHLY, nil
	case cfg.PriceIDPlusYearly:
		return v1API.Plan_UNPAPER_PLUS_YEARLY, nil
	default:
		return 0, fmt.Errorf("failed to convert stripe priceID to PB Plan: %q", priceID)
	}
}

// stripePriceIDToPBPlan converts stripe price id to protobuf plan enum
func stripePriceIDToPGPlan(cfg *config.Config, priceID string) (customers.SubscriptionPlan, error) {
	switch priceID {
	case cfg.PriceIDFree:
		return customers.SubscriptionPlanFree, nil
	case cfg.PriceIDPlusMonthly:
		return customers.SubscriptionPlanPlusMonthly, nil
	case cfg.PriceIDPlusYearly:
		return customers.SubscriptionPlanPlusYearly, nil
	default:
		return "", fmt.Errorf("failed to convert stripe priceID to PG Plan: %q", priceID)
	}
}

// StripeSubscriptionToPB converts a stripe subscription to protobuf
// dynamically adding custom fields
func StripeSubscriptionToPB(cfg *config.Config, params StripeSubscriptionToPBParams) (*v1API.Subscription, error) {
	pbSubStatus, err := stripeSubscriptionStatusToPB(params.Subscription.Status)
	if err != nil {
		return nil, err
	}
	if len(params.Subscription.Items.Data) != 1 {
		return nil, fmt.Errorf("invalid subscription items length. got: %d, want: %d", len(params.Subscription.Items.Data), 1)
	}

	subsc := &v1API.Subscription{
		Id:               params.Subscription.ID,
		CustomerId:       params.Subscription.Customer.ID,
		CurrentPeriodEnd: params.Subscription.CurrentPeriodEnd,
		Status:           pbSubStatus,
		Price: &v1API.Price{
			Id:     params.Subscription.Items.Data[0].Price.ID,
			Active: params.Subscription.Items.Data[0].Price.Active,
		},
	}

	if params.Options.WithLatestInvoice {
		latestInvoice, err := stripeLatestInvoiceToPB(params.Subscription.LatestInvoice)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert stripe latest invoice to pb: %v", err)
		}

		// Attach latest invoice
		subsc.LatestInvoice = latestInvoice
	}

	if params.Options.WithItems {
		subItem := &v1API.SubscriptionItem{
			Id: params.Subscription.Items.Data[0].ID,
		}

		// Attach Subscription Item
		subsc.Items = []*v1API.SubscriptionItem{subItem}
	}

	if params.Options.WithPlan {
		pbPlan, err := stripePriceIDToPBPlan(cfg, params.Subscription.Items.Data[0].Price.ID)
		if err != nil {
			return nil, err
		}
		subsc.Price.Plan = pbPlan
	}

	return subsc, nil
}

func stripeLatestInvoiceToPB(i *stripe.Invoice) (*v1API.Invoice, error) {
	if i == nil {
		return nil, nil
	}

	pbInvoice := &v1API.Invoice{
		Id: i.ID,
	}

	// If the user subscribed to the free tier, without adding a payment method,
	// no payment intent is attached to the latest invoice
	if i.PaymentIntent != nil {
		pbPaymentIntentStatus, err := StripePaymentIntentStatusToPB(i.PaymentIntent.Status)
		if err != nil {
			return nil, err
		}
		pbPI := &v1API.PaymentIntent{
			Id:           i.PaymentIntent.ID,
			ClientSecret: i.PaymentIntent.ClientSecret,
			Status:       pbPaymentIntentStatus,
		}

		pbInvoice.PaymentIntent = pbPI
	}

	return pbInvoice, nil
}
