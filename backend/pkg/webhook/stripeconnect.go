package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/DagDigg/unpaper/backend/customers"
	"github.com/DagDigg/unpaper/backend/pkg/stripeservice"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/sub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrMissingAcctID error returned when account id is missing
	ErrMissingAcctID = status.Error(codes.Internal, "service is missing account id")
	// ErrMissingRoomID error returned when room id is missing in metadata
	ErrMissingRoomID = status.Error(codes.Internal, "a room subscription must contain room id in metadata")
)

// HandleAccountUpdate is an handler for the webhook 'account.updated' event
func (s *Stripe) HandleAccountUpdate(data json.RawMessage) (*empty.Empty, error) {
	var acct *stripe.Account
	if err := json.Unmarshal(data, &acct); err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "failed to unmarshal stripe webhook account: %q", err)
	}
	customersDir := customers.NewDirectory(s.db)

	// Update `can_receive_payments` column based on account current status
	_, err := customersDir.UpdateConnectedAccountCanReceivePayments(s.ctx, &customers.UpdateConnectedAccountCanReceivePaymentsParams{
		AccountID:          acct.ID,
		CanReceivePayments: canAccountReceivePayments(acct),
	})
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "failed to update has_connected_account: %q", err)
	}

	return new(empty.Empty), nil
}

// HandleInvoicePaid handles 'invoice.paid' stripe webhook event
func (s *Stripe) HandleInvoicePaid(data json.RawMessage) (*empty.Empty, error) {
	// Unmarshal invoice
	res, err := unmarshalInvoiceWithSubscription(data, s.acctID)
	if err != nil {
		return new(empty.Empty), err
	}
	if s.acctID == "" {
		// Room subscriptions live in the connected account
		return new(empty.Empty), ErrMissingAcctID
	}

	if res.Subscription != nil {
		roomIDStr, ok := res.Subscription.Metadata[stripeservice.StripeMDPayForRoomAccess]
		if !ok {
			return new(empty.Empty), ErrMissingRoomID
		}

		// Invoice is for a room subscription purchase
		return storeRoomInvoice(s.svc, &stripeservice.StoreRoomSubscriptionParams{
			Subscription: res.Subscription,
			Invoice:      res.Invoice,
			RoomID:       roomIDStr,
			AccountID:    s.acctID,
		})
	}

	return new(empty.Empty), nil
}

// HandleInvoicePaymentFailed handles 'invoice.payment_failed' stripe webhook event
func (s *Stripe) HandleInvoicePaymentFailed(data json.RawMessage) (*empty.Empty, error) {
	// Unmarshal invoice
	res, err := unmarshalInvoiceWithSubscription(data, s.acctID)
	if err != nil {
		return new(empty.Empty), err
	}
	if s.acctID == "" {
		// Room subscriptions live in the connected account
		return new(empty.Empty), ErrMissingAcctID
	}

	// TODO: maybe it should send an email
	// asking to come back to the platform and complete the payment

	if res.Subscription != nil {
		roomIDStr, ok := res.Subscription.Metadata[stripeservice.StripeMDPayForRoomAccess]
		if !ok {
			return new(empty.Empty), ErrMissingRoomID
		}
		// Invoice is for a room subscription purchase
		return storeRoomInvoice(s.svc, &stripeservice.StoreRoomSubscriptionParams{
			Subscription: res.Subscription,
			Invoice:      res.Invoice,
			RoomID:       roomIDStr,
			AccountID:    s.acctID,
		})
	}

	return new(empty.Empty), nil
}

// HandleInvoicePaymentActionRequired handles 'invoice.payment_action_required' stripe webhook event
func (s *Stripe) HandleInvoicePaymentActionRequired(data json.RawMessage) (*empty.Empty, error) {
	// Unmarshal invoice
	res, err := unmarshalInvoiceWithSubscription(data, s.acctID)
	if err != nil {
		return new(empty.Empty), err
	}

	// TODO: maybe it should send an email
	// asking to come back to the platform and complete the payment

	if res.Subscription != nil {
		roomIDStr, ok := res.Subscription.Metadata[stripeservice.StripeMDPayForRoomAccess]
		if !ok {
			return new(empty.Empty), ErrMissingRoomID
		}

		// Invoice is for a room subscription purchase
		return storeRoomInvoice(s.svc, &stripeservice.StoreRoomSubscriptionParams{
			Subscription: res.Subscription,
			Invoice:      res.Invoice,
			RoomID:       roomIDStr,
			AccountID:    s.acctID,
		})
	}

	return new(empty.Empty), nil
}

func storeRoomInvoice(svc *stripeservice.Service, p *stripeservice.StoreRoomSubscriptionParams) (*empty.Empty, error) {
	_, err := svc.StoreRoomSubscription(p)
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "could not store room subscription on db: %v", err)
	}

	return new(empty.Empty), nil
}

func updateRoomSubscription(ctx context.Context, subsc *stripe.Subscription, stripesvc *stripeservice.Service, acctID string) (*empty.Empty, error) {
	// Can't fail
	roomID, _ := subsc.Metadata[stripeservice.StripeMDPayForRoomAccess]
	_, err := stripesvc.StoreRoomSubscription(&stripeservice.StoreRoomSubscriptionParams{
		Subscription: subsc,
		RoomID:       roomID,
		Options: &stripeservice.StripeSubscriptionOptions{
			WithLatestInvoice: true,
			WithItems:         true,
		},
		AccountID: acctID,
	})
	if err != nil {
		return new(empty.Empty), err
	}

	return new(empty.Empty), nil
}

// HandlePaymentIntentFailed handles 'payment_intent.payment_failed' stripe webhook event
func (s *Stripe) HandlePaymentIntentFailed(data json.RawMessage) (*empty.Empty, error) {
	var charge stripe.PaymentIntent
	err := json.Unmarshal(data, &charge)
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "error unmarshaling payment intent: %q", err)
	}

	// TODO: do something
	return new(empty.Empty), nil
}

// HandleSubscriptionUpdate handles 'customer.subscription.updated' stripe webhook event
func (s *Stripe) HandleSubscriptionUpdate(data json.RawMessage) (*empty.Empty, error) {
	var subsc *stripe.Subscription
	if err := json.Unmarshal(data, &subsc); err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "failed to unmarshal stripe webhook subscription: %q", err)
	}

	_, ok := subsc.Metadata[stripeservice.StripeMDPayForRoomAccess]
	if !ok {
		return new(empty.Empty), ErrMissingRoomID
	}

	// Subscription update is for room
	return updateRoomSubscription(s.ctx, subsc, s.svc, s.acctID)
}

// HandlePaymentIntentSucceeded is an handler for the webhook 'payment_intent.succeeded' event
func (s *Stripe) HandlePaymentIntentSucceeded(data json.RawMessage) (*empty.Empty, error) {
	var whPI *stripe.PaymentIntent
	if err := json.Unmarshal(data, &whPI); err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "failed to unmarshal stripe webhook payment intent: %q", err)
	}

	customersDir := customers.NewDirectory(s.db)

	roomID, ok := whPI.Metadata[stripeservice.StripeMDPayForRoomAccess]
	if ok {
		// Payment intent is for room access
		// TODO: do something (send email etc)
		return storeRoomPaymentIntent(&storeRoomPaymentIntentParams{
			ctx:          s.ctx,
			pi:           whPI,
			roomID:       roomID,
			customersDir: customersDir,
			acctID:       s.acctID,
		})
	}

	return new(empty.Empty), nil
}

// unmarshalInvoiceWithSubscription unmarshals a stripe invoice, fetches the most recent subscription
// attached to it, and returns both. A check on the returned subscription must be made since it can be nil
func unmarshalInvoiceWithSubscription(rawInvoice json.RawMessage, acctID string) (*InvoiceWithSubscription, error) {
	var res = &InvoiceWithSubscription{}

	// Unmarshal invoice
	var invoice = &stripe.Invoice{}
	err := json.Unmarshal(rawInvoice, invoice)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error unmarshaling invoice json: %q", err)
	}

	if invoice.PaymentIntent != nil {
		// Retrieve up-to-date payment intent
		params := &stripe.PaymentIntentParams{}
		params.SetStripeAccount(acctID)
		pi, err := paymentintent.Get(invoice.PaymentIntent.ID, params)
		if err != nil {
			return nil, err
		}
		invoice.PaymentIntent = pi
	}

	// Attach invoice
	res.Invoice = invoice
	if invoice.Subscription == nil {
		// If subscription is missing, that's ok. Return only the invoice
		return res, nil
	}

	params := &stripe.SubscriptionParams{}
	// Operation must be executed on behalf of connected account
	params.SetStripeAccount(acctID)

	subscription, err := sub.Get(invoice.Subscription.ID, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve subscription from webhook: %v", err)
	}

	// Attach subscription
	res.Subscription = subscription

	return res, nil
}

type storeRoomPaymentIntentParams struct {
	pi           *stripe.PaymentIntent
	roomID       string
	customersDir *customers.Directory
	ctx          context.Context
	acctID       string
}

func storeRoomPaymentIntent(p *storeRoomPaymentIntentParams) (*empty.Empty, error) {
	// Payment intent is for accessing a room
	userID, ok := p.pi.Metadata[stripeservice.StripeMDUserID]
	if !ok {
		// userID missing, do something
		return new(empty.Empty), status.Error(codes.Internal, "missing user id")
	}

	_, err := p.customersDir.StoreRoomSubscription(p.ctx, &customers.StoreRoomSubscriptionParams{
		ID:                   uuid.NewString(),
		UserID:               userID,
		CurrentPeriodEnd:     sql.NullTime{Time: time.Now(), Valid: false}, // One time payments have no expiry
		RoomSubscriptionType: string(customers.RoomSubscriptionTypeOneTime),
		Status:               string(customers.SubscriptionStatusActive), // One time payments are always active from the moment the payment is successful
		RoomID:               p.roomID,
		CustomerID:           p.pi.Customer.ID,
		AccountID:            p.acctID,
	})
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "failed to update user ids who paid: %v", err)
	}

	return new(empty.Empty), nil
}

func canAccountReceivePayments(a *stripe.Account) bool {
	return a.Requirements.CurrentlyDue == nil && a.ChargesEnabled && a.DetailsSubmitted
}
