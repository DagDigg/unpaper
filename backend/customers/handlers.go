package customers

import (
	"encoding/json"
	"fmt"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/core/config"
)

func pgCustomerToPB(cus Customer) (*v1API.Customer, error) {
	return &v1API.Customer{
		Id:         cus.ID,
		CustomerId: cus.CustomerID,
		FirstName:  cus.FirstName,
		LastName:   cus.LastName,
		AccountId:  cus.AccountID.String,
	}, nil
}

func pgSubscriptionToPB(sub StripeSubscription) (*v1API.Subscription, error) {
	status, err := pgSubscriptionStatusToPB(SubscriptionStatus(sub.Status))
	if err != nil {
		return nil, err
	}
	latestInvoice := &v1API.Invoice{}
	if err = json.Unmarshal(sub.LatestInvoice, latestInvoice); err != nil {
		return nil, err
	}

	return &v1API.Subscription{
		Id:               sub.ID,
		CustomerId:       sub.CustomerID,
		CurrentPeriodEnd: sub.CurrentPeriodEnd.Unix(),
		Status:           status,
		LatestInvoice:    latestInvoice,
	}, nil
}

func pgRoomSubscriptionToPB(sub RoomSubscription) (*v1API.RoomSubscription, error) {
	status, err := pgSubscriptionStatusToPB(SubscriptionStatus(sub.Status))
	if err != nil {
		return nil, err
	}
	roomSubscType, err := pgRoomSubscriptionTypeToPB(RoomSubscriptionType(sub.RoomSubscriptionType))
	if err != nil {
		return nil, err
	}
	var currPeriodEnd int64
	if sub.CurrentPeriodEnd.Valid {
		currPeriodEnd = sub.CurrentPeriodEnd.Time.Unix()
	}
	latestInvoice := &v1API.Invoice{}
	if err := json.Unmarshal(sub.LatestInvoice, latestInvoice); err != nil {
		return nil, err
	}

	return &v1API.RoomSubscription{
		Id:                   sub.ID,
		CustomerId:           sub.CustomerID,
		CurrentPeriodEnd:     currPeriodEnd,
		Status:               status,
		RoomId:               sub.RoomID,
		RoomSubscriptionType: roomSubscType,
		LatestInvoice:        latestInvoice,
	}, nil

}

func pgPriceToPB(price StripePrice) (*v1API.Price, error) {
	plan, err := pgPlanToPB(SubscriptionPlan(price.Plan))
	if err != nil {
		return nil, err
	}

	return &v1API.Price{
		Id:     price.ID,
		Active: price.Active,
		Plan:   plan,
	}, nil
}

func pgSubscriptionStatusToPB(status SubscriptionStatus) (v1API.SubscriptionStatus, error) {
	switch status {
	case SubscriptionStatusActive:
		return v1API.SubscriptionStatus_ACTIVE, nil
	case SubscriptionStatusCanceled:
		return v1API.SubscriptionStatus_CANCELED, nil
	case SubscriptionStatusIncomplete:
		return v1API.SubscriptionStatus_INCOMPLETE, nil
	case SubscriptionStatusIncompleteExpired:
		return v1API.SubscriptionStatus_INCOMPLETE_EXPIRED, nil
	case SubscriptionStatusPastDue:
		return v1API.SubscriptionStatus_PAST_DUE, nil
	case SubscriptionStatusTrialing:
		return v1API.SubscriptionStatus_TRIALING, nil
	case SubscriptionStatusUnpaid:
		return v1API.SubscriptionStatus_UNPAID, nil
	default:
		return 0, fmt.Errorf("could not parse subscription status from proto to pg")
	}
}

// getPriceIDByPlan returns the env priceID mapping value for
// the provided protobuf plan
func getPriceIDByPlan(cfg *config.Config, plan v1API.Plan) (string, error) {
	switch plan {
	case v1API.Plan_UNPAPER_FREE:
		return cfg.PriceIDFree, nil
	case v1API.Plan_UNPAPER_PLUS_MONTHLY:
		return cfg.PriceIDPlusMonthly, nil
	case v1API.Plan_UNPAPER_PLUS_YEARLY:
		return cfg.PriceIDPlusYearly, nil
	default:
		return "", fmt.Errorf("invalid plan: %q", plan)
	}
}

// pgPlanToPB converts postgres plan enum to protobuf
func pgPlanToPB(plan SubscriptionPlan) (v1API.Plan, error) {
	switch plan {
	case SubscriptionPlanFree:
		return v1API.Plan_UNPAPER_FREE, nil
	case SubscriptionPlanPlusMonthly:
		return v1API.Plan_UNPAPER_PLUS_MONTHLY, nil
	case SubscriptionPlanPlusYearly:
		return v1API.Plan_UNPAPER_PLUS_YEARLY, nil
	default:
		return 0, fmt.Errorf("invalid plan received: %q", plan)
	}
}

// pgPaymentMethodToPB converts postgres payment_method to protobuf
func pgPaymentMethodToPB(pm StripeDefaultPaymentMethod) (*v1API.PaymentMethod, error) {
	return &v1API.PaymentMethod{
		Id:        pm.ID,
		UserId:    pm.UserID,
		LastFour:  pm.LastFour,
		ExpMonth:  pm.ExpMonth,
		ExpYear:   pm.ExpYear,
		IsDefault: pm.IsDefault.Bool,
	}, nil
}

func pgRoomSubscriptionTypeToPB(t RoomSubscriptionType) (v1API.RoomSubcriptionType_Enum, error) {
	switch t {
	case RoomSubscriptionTypeOneTime:
		return v1API.RoomSubcriptionType_ONE_TIME, nil
	case RoomSubscriptionTypeSubscriptionMonthly:
		return v1API.RoomSubcriptionType_SUBSCRIPTION_MONTHLY, nil
	default:
		return 0, fmt.Errorf("invalid subscription type received: %v", t)
	}
}

// pgConnectedCustomerToPB converts a postgres connected customer to PB
func pgConnectedCustomerToPB(c ConnectedCustomer) *v1API.ConnectedCustomer {
	return &v1API.ConnectedCustomer{
		UserId:              c.UserID,
		CustomerId:          c.CustomerID,
		ConnectedCustomerId: c.ConnectedCustomerID,
		AccountId:           c.AccountID,
	}
}

// pgConnectedAccountToPB converts a postgres connected account to PB
func pgConnectedAccountToPB(a ConnectedAccount) *v1API.ConnectedAccount {
	return &v1API.ConnectedAccount{
		UserId:             a.UserID,
		CustomerId:         a.CustomerID,
		AccountId:          a.AccountID,
		CanReceivePayments: a.CanReceivePayments,
	}
}
