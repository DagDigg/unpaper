package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/smtp"

	"github.com/DagDigg/unpaper/backend/customers"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/stripeservice"
	"github.com/DagDigg/unpaper/core/config"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stripe/stripe-go/v72"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InvoiceWithSubscription struct
// used for wrapping a stripe invoice and subscription
type InvoiceWithSubscription struct {
	Invoice *stripe.Invoice
	// Most up to date subscription attached
	// to the invoice, if any
	Subscription *stripe.Subscription
}

type handlePlatformInvoiceParams struct {
	cfg          *config.Config
	ctx          context.Context
	customersDir *customers.Directory
	inWithSub    *InvoiceWithSubscription
}

func handlePlatformInvoice(p *handlePlatformInvoiceParams) (*empty.Empty, error) {
	_, err := p.customersDir.UpdateSubscriptionStatus(p.ctx, customers.UpdateSubscriptionStatusParams{
		ID:     p.inWithSub.Subscription.ID,
		Status: string(customers.SubscriptionStatusActive),
	})
	if err != nil {
		return new(empty.Empty), fmt.Errorf("error updating subscription status on db: %q", err)
	}

	// Send emails only on prod environment
	if p.cfg.Environment == "dev" {
		return new(empty.Empty), nil
	}

	// Do not send email invoice if user subscribed to free plan
	if p.inWithSub.Invoice.Lines.Data[0].Price.ID == p.cfg.PriceIDFree {
		return new(empty.Empty), nil
	}

	msg := []byte("To: " + "dontcare@gmail.com" + "\r\n" +
		"Subject: Your invoice\r\n" +
		"\r\n" +
		"Here you can find your invoice: " + p.inWithSub.Invoice.InvoicePDF)

	auth := smtp.PlainAuth("", p.cfg.SMTPUser, p.cfg.SMTPPass, p.cfg.SMTPDomain)
	err = smtp.SendMail(p.cfg.SMTPDomain+":"+p.cfg.SMTPPort, auth, "unpaper@me.com", []string{"dontcare@gmail.com"}, msg)
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "could not send email: '%v'", err)
	}

	return new(empty.Empty), nil
}

func updatePlatformSubscription(ctx context.Context, subsc *stripe.Subscription, stripesvc *stripeservice.Service) (*empty.Empty, error) {
	if subsc.Status == stripe.SubscriptionStatusCanceled {
		// Latest retry failed. Set subscription back to free
		freeSubsc, err := stripesvc.UpdatePlatformSubscription(&stripeservice.UpdatePlatformSubscriptionParams{
			Customer: subsc.Customer,
			Plan:     v1API.Plan_UNPAPER_FREE,
		})
		if err != nil {
			return new(empty.Empty), status.Errorf(codes.Internal, "failed to revert to free subscription: %v", err)
		}

		subsc = freeSubsc
	}

	_, err := stripesvc.StorePlatformSubscription(&stripeservice.StorePlatformSubscriptionParams{
		Subscription: subsc,
	})
	return new(empty.Empty), err
}

// HandleSubscriptionDeleted is an handler for the webhook 'customer.subscription.deleted' event
func (s *Stripe) HandleSubscriptionDeleted(data json.RawMessage) (*empty.Empty, error) {
	var whSubsc *stripe.Subscription
	if err := json.Unmarshal(data, &whSubsc); err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "failed to unmarshal stripe webhook subscription: %q", err)
	}
	userID, ok := whSubsc.Metadata[stripeservice.StripeMDUserID]
	if !ok {
		return new(empty.Empty), status.Error(codes.Internal, "missing userID in metadata")
	}
	subsc, err := s.svc.CreatePlatformSubscription(whSubsc.Customer.ID, userID, s.cfg.PriceIDFree)
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "failed to create free subscription: %v", err)
	}

	_, err = s.svc.StoreSubscription(subsc, userID)
	if err != nil {
		return new(empty.Empty), status.Error(codes.Internal, err.Error())
	}

	return new(empty.Empty), err
}

// HandlePMAttached  is an handler for the webhook 'payment_method.attached' event
func (s *Stripe) HandlePMAttached(data json.RawMessage) (*empty.Empty, error) {
	var whPM *stripe.PaymentMethod
	if err := json.Unmarshal(data, &whPM); err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "failed to unmarshal stripe webhook payment method: %q", err)
	}

	customersDir := customers.NewDirectory(s.db)

	userID, err := customersDir.GetCustomerUserID(s.ctx, whPM.Customer.ID)
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "failed to get userid from customer id %q : %q", whPM.Customer.ID, err)
	}

	_, err = customersDir.StoreDefaultPM(s.ctx, &customers.StoreDefaultPaymentMethodParams{
		ID:         whPM.ID,
		UserID:     userID,
		CustomerID: whPM.Customer.ID,
		LastFour:   whPM.Card.Last4,
		ExpMonth:   int32(whPM.Card.ExpMonth),
		ExpYear:    int32(whPM.Card.ExpYear),
	})
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "failed to update default pm: %q", err)
	}

	// TODO: DO NOT ATTACH PM ON CONNECTED CUSTOMER CREATION, BUT ONLY ON WEBHOOK
	// Sync new pm with all connected accounts where this customer lives
	ctx := context.Background()
	connCusData, err := customersDir.GetAllConnectCustomers(ctx, whPM.Customer.ID)
	if err != nil {
		if err != sql.ErrNoRows {
			return new(empty.Empty), status.Errorf(codes.Internal, "error retrieving connected customers data: %v", err)
		}
	}
	if connCusData != nil {
		stripesvc := stripeservice.New(ctx, s.db, s.cfg)
		for _, data := range connCusData {
			_, err := stripesvc.SyncConnectPaymentMethods(stripeservice.SyncConnectPaymentMethodsParams{
				CustomerID:          whPM.Customer.ID,
				ConnectedCustomerID: data.ConnectedCustomerID,
				AccountID:           data.AccountID,
			})
			if err != nil {
				// TODO: Do something better. This print is because we dont want to block every other sync
				fmt.Printf("error synching platform customer (%s) pm on account id %q: %v", whPM.Customer.ID, data.AccountID, err)
			}
		}

	}

	return new(empty.Empty), nil
}
