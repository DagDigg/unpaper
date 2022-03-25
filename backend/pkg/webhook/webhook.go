package webhook

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/DagDigg/unpaper/backend/pkg/stripeservice"
	"github.com/DagDigg/unpaper/core/config"
	"github.com/golang/protobuf/ptypes/empty"
)

// Stripe is a webhook handler for stripe events
type Stripe struct {
	ctx    context.Context
	db     *sql.DB
	cfg    *config.Config
	svc    *stripeservice.Service
	acctID string
}

// StripeWebhook interface that a handler must implement
// in order to reply to incoming webhooks events
type StripeWebhook interface {

	// Payment Method
	HandlePMAttached(data json.RawMessage) (*empty.Empty, error)
}

// StripeConnectWebhook that a handler must
// implement in order to reply to incoming stripe
// connect webhook events
type StripeConnectWebhook interface {
	// Account
	HandleAccountUpdate(data json.RawMessage) (*empty.Empty, error)

	// Invoice
	HandleInvoicePaid(data json.RawMessage) (*empty.Empty, error)
	HandleInvoicePaymentFailed(data json.RawMessage) (*empty.Empty, error)
	HandleInvoicePaymentActionRequired(data json.RawMessage) (*empty.Empty, error)

	// Subscription
	HandleSubscriptionDeleted(data json.RawMessage) (*empty.Empty, error)
	HandleSubscriptionUpdate(data json.RawMessage) (*empty.Empty, error)

	// Payment Intent
	HandlePaymentIntentSucceeded(data json.RawMessage) (*empty.Empty, error)
	HandlePaymentIntentFailed(data json.RawMessage) (*empty.Empty, error)
}

// NewStripeParams parameters for creating a new
// Stripe webhook handler
type NewStripeParams struct {
	Ctx           context.Context
	DB            *sql.DB
	Cfg           *config.Config
	StripeService *stripeservice.Service
	AccountID     string
}

// NewStripe returns a new Stripe webhook handler
func NewStripe(p *NewStripeParams) StripeWebhook {
	return &Stripe{
		ctx: p.Ctx,
		db:  p.DB,
		cfg: p.Cfg,
		svc: p.StripeService,
	}
}

func NewStripeConnect(p *NewStripeParams) StripeConnectWebhook {
	return &Stripe{
		ctx:    p.Ctx,
		db:     p.DB,
		cfg:    p.Cfg,
		svc:    p.StripeService,
		acctID: p.AccountID,
	}
}
