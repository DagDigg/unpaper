package v1

import (
	"context"
	"fmt"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/mdutils"
	"github.com/DagDigg/unpaper/backend/pkg/stripeservice"
	webhookhandler "github.com/DagDigg/unpaper/backend/pkg/webhook"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stripe/stripe-go/v72/webhook"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// getSignatureFromCtx retrieves stripe signature from context metadata
func getSignatureFromCtx(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.InvalidArgument, "missing metadata")
	}
	stripeSignature, ok := mdutils.GetFirstMDValue(md, "stripe-signature")
	if !ok {
		return "", status.Error(codes.Internal, "missing stripe-signature in metadata")
	}

	return stripeSignature, nil
}

// StripeWebhook handler for incoming stripe events
func (s *unpaperServiceServer) StripeWebhook(ctx context.Context, req *v1API.StripeWebhookRequest) (*empty.Empty, error) {
	stripeSignature, err := getSignatureFromCtx(ctx)
	if err != nil {
		return new(empty.Empty), err
	}

	stripesvc := stripeservice.New(ctx, s.db, s.cfg)

	// https://stripe.com/docs/webhooks/signatures#verify-official-libraries
	endpointSecret := "my_stripe_webhook_secret"
	event, err := webhook.ConstructEvent(req.GetRaw(), stripeSignature, endpointSecret)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error constructing webhook event: %v", err)
	}
	handler := webhookhandler.NewStripe(&webhookhandler.NewStripeParams{
		Ctx:           ctx,
		DB:            s.db,
		Cfg:           s.cfg,
		StripeService: stripesvc,
	})

	fmt.Printf("\n\nType: %v, Acct: %v\n\n", event.Type, event.Account)
	switch event.Type {
	case "payment_method.attached":
		return handler.HandlePMAttached(event.Data.Raw)
	}

	return new(empty.Empty), nil
}

func (s *unpaperServiceServer) StripeConnectWebhook(ctx context.Context, req *v1API.StripeWebhookRequest) (*empty.Empty, error) {
	stripeSignature, err := getSignatureFromCtx(ctx)
	if err != nil {
		return new(empty.Empty), err
	}

	stripesvc := stripeservice.New(ctx, s.db, s.cfg)

	// https://stripe.com/docs/webhooks/signatures#verify-official-libraries
	endpointSecret := "my_stripe_webhook_secret"
	event, err := webhook.ConstructEvent(req.GetRaw(), stripeSignature, endpointSecret)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error constructing webhook event: %v", err)
	}
	handler := webhookhandler.NewStripeConnect(&webhookhandler.NewStripeParams{
		Ctx:           ctx,
		DB:            s.db,
		Cfg:           s.cfg,
		StripeService: stripesvc,
		AccountID:     event.Account,
	})

	fmt.Printf("\n\nCONNECT Type: %v, Acct: %v\n\n", event.Type, event.Account)
	switch event.Type {
	case "account.updated":
		return handler.HandleAccountUpdate(event.Data.Raw)
	case "invoice.paid":
		// Used to provision services after the trial has ended.
		// The status of the invoice will show up as paid. Store the status in your
		// database to reference when a user accesses your service to avoid hitting rate
		// limits.
		return handler.HandleInvoicePaid(event.Data.Raw)
	case "invoice.payment_failed":
		// If the payment fails or the customer does not have a valid payment method,
		// an invoice.payment_failed event is sent, the subscription becomes past_due.
		// Use this webhook to notify your user that their payment has
		// failed and to retrieve new card details.
		return handler.HandleInvoicePaymentFailed(event.Data.Raw)
	case "invoice.payment_action_required":
		//Some payment methods may require additional steps, such as customer authentication, to complete.
		// When an invoice’s payment requires additional action, invoice.payment_action_required
		// and invoice.payment_failed events are sent and the status of the PaymentIntent is requires_action
		return handler.HandleInvoicePaymentActionRequired(event.Data.Raw)
	case "customer.subscription.deleted":
		// handle subscription cancelled automatically based
		// upon your subscription settings. Or if the user cancels it.
		return handler.HandleSubscriptionDeleted(event.Data.Raw)
	case "payment_intent.payment_failed":
		return handler.HandlePaymentIntentFailed(event.Data.Raw)
	case "customer.subscription.updated":
		// Occurs whenever a subscription changes (e.g., switching from one plan to another,
		//or changing the status from trial to active).
		return handler.HandleSubscriptionUpdate(event.Data.Raw)
	case "payment_intent.succeeded":
		return handler.HandlePaymentIntentSucceeded(event.Data.Raw)
	}
	return new(empty.Empty), nil
}
