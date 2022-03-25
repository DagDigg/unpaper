package stripeservice

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DagDigg/unpaper/backend/customers"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/core/config"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/invoice"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/paymentmethod"
	"github.com/stripe/stripe-go/v72/product"
	"github.com/stripe/stripe-go/v72/sub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service contains methods for interacting
// with the stripe api and platform database
type Service struct {
	db  *sql.DB
	ctx context.Context
	cfg *config.Config
}

// New returns a new stripeservice
func New(ctx context.Context, db *sql.DB, cfg *config.Config) *Service {
	return &Service{
		db:  db,
		ctx: ctx,
		cfg: cfg,
	}
}

const (
	// StripeMDUserID user_id stripe metadata
	StripeMDUserID = "stripe_md_user_id"
	// StripeMDPayForRoomAccess room id stripe metadata
	StripeMDPayForRoomAccess = "pay_for_room_access"
)

// CreateCustomer creates a new Stripe customer
func (s *Service) CreateCustomer(p *CreateCustomerParams) (*stripe.Customer, error) {
	fullName := strings.Trim(p.GivenName+" "+p.FamilyName, " ")
	params := &stripe.CustomerParams{
		Description: stripe.String("GogoCrowd platform customer"),
		Email:       stripe.String(p.Email),
		Name:        stripe.String(fullName),
	}

	c, err := customer.New(params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create stripe customer: %q", err)
	}

	return c, nil
}

// CreateConnectedCustomer creates a stripe customer on the specified account id.
// The customer must already exist on the platform and must have a default payment method attached
func (s *Service) CreateConnectedCustomer(p *CreateConnectedCustomerParams) (*stripe.Customer, error) {
	// Fetch up to date stripe platform customer
	platformCus, err := customer.Get(p.CustomerID, nil)
	if err != nil {
		return nil, err
	}
	// Platform customer must have a default PM
	if platformCus.InvoiceSettings.DefaultPaymentMethod == nil {
		return nil, status.Error(codes.Internal, "customer has no default payment method")
	}

	// Create payment method on connected account
	pmParams := &stripe.PaymentMethodParams{
		Customer:      stripe.String(platformCus.ID),
		PaymentMethod: stripe.String(platformCus.InvoiceSettings.DefaultPaymentMethod.ID),
	}
	pmParams.SetStripeAccount(p.AccountID)

	pm, err := paymentmethod.New(pmParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create payment method on connected account: %v", err)
	}

	params := &stripe.CustomerParams{
		PaymentMethod: stripe.String(pm.ID),
		Name:          stripe.String(platformCus.Name),
		Email:         stripe.String(platformCus.Email),
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm.ID),
		},
	}
	params.SetStripeAccount(p.AccountID)

	// Create connected customer
	connCus, err := customer.New(params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create connected customer: %v", err)
	}

	return connCus, nil
}

// StoreCustomer saves a Stripe customer on db, converting it to the protobuf version
func (s *Service) StoreCustomer(cus *stripe.Customer, p StoreCustomerParams) (*v1API.Customer, error) {
	customersDir := customers.NewDirectory(s.db)

	// Store stripe customer on db
	pbCus, err := customersDir.CreateCustomer(s.ctx, &customers.CreateCustomerParams{
		ID:         p.UserID,
		CustomerID: cus.ID,
		FirstName:  p.GivenName,
		LastName:   p.FamilyName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store customer: %v", err)
	}

	return pbCus, nil
}

// CreatePlatformSubscription creates a free Stripe subscription
func (s *Service) CreatePlatformSubscription(customerID, userID, priceID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Plan: stripe.String(priceID),
			},
		},
	}
	params.AddMetadata(StripeMDUserID, userID)

	return sub.New(params)
}

// StoreSubscription saves the stripe subscription on database adapting to protobuf
func (s *Service) StoreSubscription(subsc *stripe.Subscription, userID string) (*v1API.Subscription, error) {
	customersDir := customers.NewDirectory(s.db)

	if len(subsc.Items.Data) == 0 {
		return nil, status.Errorf(codes.Internal, "missing subscription items")
	}

	// Store stripe subscription and price on db
	return customersDir.StoreSubscriptionWithPrice(s.ctx, &customers.StoreSubscriptionWithPriceParams{
		Subscription: &customers.StoreSubscriptionParams{
			ID:               subsc.ID,
			UserID:           userID,
			CustomerID:       subsc.Customer.ID,
			CurrentPeriodEnd: time.Unix(subsc.CurrentPeriodEnd, 0),
			// Safe cast since customers.SubscriptionStatus is derived from stripe's subscription.Status
			Status: string(customers.SubscriptionStatus(subsc.Status)),
		},
		Price: &customers.StorePriceParams{
			ID:         subsc.Items.Data[0].Price.ID,
			UserID:     userID,
			CustomerID: subsc.Customer.ID,
			Active:     subsc.Plan.Active,
			Plan:       string(customers.SubscriptionPlanFree),
		},
	})
}

// CreatePaymentIntent creates a connect payment intent applying the fee
func (s *Service) CreatePaymentIntent(p *CreateConnectPaymentIntentParams) (*stripe.PaymentIntent, error) {
	feeAmt := calcFeeAmount(float64(p.Amount), p.PlatformFeePercent)

	// Retrieve up-to-date customer
	cusParams := &stripe.CustomerParams{}
	cusParams.AddExpand("invoice_settings.default_payment_method")
	cusParams.SetStripeAccount(p.ReceiverAccountID)
	cus, err := customer.Get(p.SenderConnectCustomerID, cusParams)
	if err != nil {
		return nil, err
	}
	// Check that customer has a default payment method.
	// This is mandatory since we're going to attach it to the payment intent
	if cus.InvoiceSettings.DefaultPaymentMethod == nil {
		return nil, fmt.Errorf("connected customer has no default payment method")
	}

	params := &stripe.PaymentIntentParams{
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		Customer:             stripe.String(p.SenderConnectCustomerID),
		Amount:               stripe.Int64(p.Amount),
		Currency:             stripe.String(string(stripe.CurrencyUSD)),
		ApplicationFeeAmount: stripe.Int64(int64(feeAmt)),
		PaymentMethod:        stripe.String(cus.InvoiceSettings.DefaultPaymentMethod.ID),
	}
	if p.Metadata != nil {
		for k, v := range p.Metadata {
			params.AddMetadata(k, v)
		}
	}
	params.SetStripeAccount(p.ReceiverAccountID)

	return paymentintent.New(params)
}

// CreateRoomSubscription creates a stripe subscription for the room
func (s *Service) CreateRoomSubscription(p *CreateRoomSubscriptionParams) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{
		Customer:              stripe.String(p.SenderConnectedCustomerID),
		ApplicationFeePercent: stripe.Float64(p.PlatformFeePct),
		TrialEnd:              stripe.Int64(time.Now().Add(20 * time.Second).Unix()), // TODO: remove
		Items: []*stripe.SubscriptionItemsParams{
			{
				PriceData: &stripe.SubscriptionItemPriceDataParams{
					UnitAmount: stripe.Int64(p.Amount),
					Currency:   stripe.String(string(stripe.CurrencyUSD)),
					Product:    stripe.String(p.ProductID),
					Recurring: &stripe.SubscriptionItemPriceDataRecurringParams{
						Interval: stripe.String(string(stripe.PriceRecurringIntervalMonth)),
					},
				},
			},
		},
	}
	params.AddExpand("latest_invoice.payment_intent")
	params.SetStripeAccount(p.ReceiverAccountID) // Execute operation on behalf of connected account

	if p.Metadata != nil {
		for k, v := range p.Metadata {
			params.AddMetadata(k, v)
		}
	}

	return sub.New(params)
}

// StoreRoomSubscription saves on database a room subscription
func (s *Service) StoreRoomSubscription(p *StoreRoomSubscriptionParams) (*v1API.RoomSubscription, error) {
	customersDir := customers.NewDirectory(s.db)
	userID, ok := p.Subscription.Metadata[StripeMDUserID]
	if !ok {
		return nil, status.Error(codes.Internal, "missing user id in metadata")
	}

	latestInvoice := p.Subscription.LatestInvoice
	if p.Invoice != nil {
		latestInvoice = p.Invoice
	}

	pbLatestInvoice, err := stripeLatestInvoiceToPB(latestInvoice)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert stripe latest invoice to pb: %v", err)
	}
	rawLatestInvoice, err := json.Marshal(pbLatestInvoice)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal latest invoice: %v", err)
	}

	// Store Room subscription
	roomSubsc, err := customersDir.StoreRoomSubscription(s.ctx, &customers.StoreRoomSubscriptionParams{
		ID:                   p.Subscription.ID,
		UserID:               userID,
		CustomerID:           p.Subscription.Customer.ID,
		CurrentPeriodEnd:     sql.NullTime{Time: time.Unix(p.Subscription.CurrentPeriodEnd, 0), Valid: true},
		RoomID:               p.RoomID,
		RoomSubscriptionType: string(customers.RoomSubscriptionTypeSubscriptionMonthly), // TODO: handle other subscription types
		LatestInvoice:        rawLatestInvoice,
		// Safe cast since customers.SubscriptionStatus is derived from stripe's subscription.Status
		Status:    string(customers.SubscriptionStatus(p.Subscription.Status)),
		AccountID: p.AccountID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error storing room subscription: %v", err)
	}

	return roomSubsc, nil
}

// StorePlatformSubscription saves on database the platform subscription
func (s *Service) StorePlatformSubscription(p *StorePlatformSubscriptionParams) (*v1API.Subscription, error) {
	customersDir := customers.NewDirectory(s.db)

	if len(p.Subscription.Items.Data) != 1 {
		return nil, status.Errorf(codes.Internal, "wrong number of subscription items. got: %q, want: %q", len(p.Subscription.Items.Data), 1)
	}

	// Retrieve userID from subscripion
	userID, ok := p.Subscription.Metadata[StripeMDUserID]
	if !ok {
		return nil, status.Error(codes.Internal, "failed to get userID from metadata")
	}

	// Build pb invoice
	pbLatestInvoice, err := stripeLatestInvoiceToPB(p.Subscription.LatestInvoice)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal latest invoice: %v", err)
	}

	plan, err := stripePriceIDToPGPlan(s.cfg, p.Subscription.Items.Data[0].Price.ID)
	if err != nil {
		return nil, err
	}
	rawLatestInvoice, err := json.Marshal(pbLatestInvoice)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal latest invoice")
	}
	pbSubsc, err := customersDir.StoreSubscriptionWithPrice(s.ctx, &customers.StoreSubscriptionWithPriceParams{
		WithTrial: true,
		Subscription: &customers.StoreSubscriptionParams{
			ID:               p.Subscription.ID,
			UserID:           userID,
			CustomerID:       p.Subscription.Customer.ID,
			CurrentPeriodEnd: time.Unix(p.Subscription.CurrentPeriodEnd, 0),
			Status:           string(customers.SubscriptionStatus(p.Subscription.Status)),
			LatestInvoice:    rawLatestInvoice,
		},
		Price: &customers.StorePriceParams{
			ID:         p.Subscription.Items.Data[0].Price.ID,
			UserID:     userID,
			CustomerID: p.Subscription.Customer.ID,
			Active:     p.Subscription.Plan.Active,
			Plan:       string(plan),
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error saving subscription on db: %q", err)
	}

	return pbSubsc, nil
}

// UpdatePlatformSubscription performs the stripe subscruption update to the
// param passed plan. It also sets the customer default payment method for the subscription invoice.
// Note: the stripe customer must have been gotten with the "invoice_settings.default_payment_method"
func (s *Service) UpdatePlatformSubscription(p *UpdatePlatformSubscriptionParams) (*stripe.Subscription, error) {
	customersDir := customers.NewDirectory(s.db)

	if len(p.Customer.Subscriptions.Data) != 1 {
		// User should always have one
		// active subscription at time
		return nil, status.Errorf(codes.Internal, "user must have one active subscription. current: %v", len(p.Customer.Subscriptions.Data))
	}
	if len(p.Customer.Subscriptions.Data[0].Items.Data) != 1 {
		// User should always have one active
		// subscription item at time
		return nil, status.Errorf(codes.Internal, "user must have one active subscription item. current: %v", len(p.Customer.Subscriptions.Data))

	}
	if p.Customer.InvoiceSettings.DefaultPaymentMethod == nil {
		// No default payment method.
		// This can occur when the customer is fetched without expanding it
		return nil, status.Errorf(codes.Internal, "missing default payment method for stripe customer")
	}

	// Retrieve trial_used from database. If the user hasn't used the trial,
	// set the subscription param trial, otherwise do not set that value
	trialUsed, err := customersDir.HasCustomerUsedTrial(context.Background(), p.Customer.ID)
	if err != nil {
		return nil, err
	}

	// Convert protobuf plan to env price id
	priceID, err := pbPlanToPriceID(s.cfg, p.Plan)
	if err != nil {
		return nil, err
	}
	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(p.Customer.Subscriptions.Data[0].Items.Data[0].ID),
				Price: stripe.String(priceID),
			},
		},
	}
	// TODO: When switching from monthly to yearly and vice-versa, trial should be preserved
	if !trialUsed {
		// If user hasn't used trial, the billing cycle anchor
		// should match with the trial_end

		// params.TrialPeriodDays = stripe.Int64(15)
		trialEnd := time.Now().Add(30 * time.Second).Unix()
		params.TrialEnd = stripe.Int64(trialEnd)
		params.ProrationBehavior = stripe.String(string(stripe.SubscriptionProrationBehaviorNone))
	}

	params.AddExpand("latest_invoice.payment_intent")

	return sub.Update(p.Customer.Subscriptions.Data[0].ID, params)
}

// GetRoomSubscriptionForUserID retrieves the most up-to-date stripe subscription
// for the user id subscribed to the room id param, if any
func (s *Service) GetRoomSubscriptionForUserID(p GetRoomSubscriptionForUserIDParams) (*stripe.Subscription, error) {
	customersDir := customers.NewDirectory(s.db)
	roomSubsc, err := customersDir.GetRoomSubscriptionForUserID(s.ctx, &customers.GetRoomSubscriptionForUserIDParams{
		RoomID: p.RoomID,
		UserID: p.UserID,
	})
	if err != nil {
		return nil, err
	}

	// Get up-to-date subscription with payment intent
	params := &stripe.SubscriptionParams{}
	params.AddExpand("latest_invoice.payment_intent")
	params.SetStripeAccount(p.AccountID)
	subsc, err := sub.Get(roomSubsc.Id, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting stripe subscription for id %q : %v", roomSubsc.Id, err)
	}

	if p.Options.StrictPaymentIntentCheck == true {
		// Strict check if latest invoice AND payment intent are not nil
		if subsc.LatestInvoice == nil {
			return nil, status.Error(codes.Internal, "missing latest invoice on stripe subscription")
		}
		if subsc.LatestInvoice.PaymentIntent == nil {
			return nil, status.Error(codes.Internal, "missing payment intent on stripe subscription")
		}
	}

	return subsc, nil
}

// RetrieveAndStoreRoomSubscriptionForUserID retrieves the most up-to-date stripe room subscription,
// stores it on database and return the protobuf converted subscription
func (s *Service) RetrieveAndStoreRoomSubscriptionForUserID(p GetRoomSubscriptionForUserIDParams) (*v1API.RoomSubscription, error) {
	stripeSubsc, err := s.GetRoomSubscriptionForUserID(GetRoomSubscriptionForUserIDParams{
		RoomID:    p.RoomID,
		UserID:    p.UserID,
		AccountID: p.AccountID,
		Options:   p.Options,
	})
	if err != nil {
		return nil, err
	}

	return s.StoreRoomSubscription(&StoreRoomSubscriptionParams{
		Subscription: stripeSubsc,
		RoomID:       p.RoomID,
		AccountID:    p.AccountID,
	})
}

// GetPlatformInvoicePreview calculates a stripe invoice preview, adding coupon if necessary
func (s *Service) GetPlatformInvoicePreview(p *GetPlatformInvoicePreviewParams) (*v1API.Invoice, error) {
	priceID, err := pbPlanToPriceID(s.cfg, p.Plan)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error converting protobuf plan to price id: %q", err)
	}

	params := &stripe.InvoiceParams{
		Customer:     stripe.String(p.CustomerID),
		Subscription: stripe.String(p.SubscriptionID),
		Coupon:       stripe.String(p.Coupon),
		SubscriptionItems: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(p.SubscriptionItemID),
				Price: stripe.String(priceID),
			},
		},
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve subscription from stripe: %q", err)
	}

	in, err := invoice.GetNext(params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve invoice preview: %q", err)
	}

	return &v1API.Invoice{
		Id:        in.ID,
		Subtotal:  in.Subtotal,
		AmountDue: in.AmountDue,
		Plan:      p.Plan,
	}, nil
}

// CreateProduct creates a new stripe product
func (s *Service) CreateProduct(p CreateProductParams) (*stripe.Product, error) {
	params := &stripe.ProductParams{
		Name: stripe.String(p.Name),
	}
	if p.AccountID != "" {
		params.SetStripeAccount(p.AccountID)
	}
	return product.New(params)
}

// SyncConnectPaymentMethods replicates the current platform customer payment methods on the connected account.
// It first detatches every connected account payment method, and then adds the ones on the platform.
// Currently only one payment method at time is supported, so it detatches the previous ones and set the
// platform default payment method as connect default one. It returns an error if the platform has no default payment method.
// On success it returns the new CONNECTED customer
func (s *Service) SyncConnectPaymentMethods(p SyncConnectPaymentMethodsParams) (*stripe.Customer, error) {
	// Retrieve up-to-date platform customer
	platCusParams := &stripe.CustomerParams{}
	platCusParams.AddExpand("invoice_settings.default_payment_method")
	platCus, err := customer.Get(p.CustomerID, platCusParams)
	if err != nil {
		return nil, err
	}
	// Retrieve up-to-date connected customer payment methods
	connCusPMParams := &stripe.PaymentMethodListParams{
		Customer: stripe.String(p.ConnectedCustomerID),
		Type:     stripe.String("card"),
	}
	connCusPMParams.SetStripeAccount(p.AccountID) // We want to retrieve the customer PMs on the connected account
	l := paymentmethod.List(connCusPMParams)

	// Detach every connected payment method
	for l.Next() {
		params := &stripe.PaymentMethodDetachParams{}
		params.SetStripeAccount(p.AccountID)
		_, err := paymentmethod.Detach(l.PaymentMethod().ID, params)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error detaching payment method from connected customer: %q", err)
		}
	}

	// Attach current platform customer payment method
	// It attaches the default one, so it is mandatory to have a default pm
	if platCus.InvoiceSettings.DefaultPaymentMethod == nil {
		return nil, status.Error(codes.Internal, "platform customer must have a default payment method")
	}
	// Create payment method on connected account
	pmParams := &stripe.PaymentMethodParams{
		Customer:      stripe.String(platCus.ID),
		PaymentMethod: stripe.String(platCus.InvoiceSettings.DefaultPaymentMethod.ID),
	}
	pmParams.SetStripeAccount(p.AccountID)
	pm, err := paymentmethod.New(pmParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error creating payment method on connected account: %v", err)
	}

	// Attach payment method to the connected account
	pmAttachParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(p.ConnectedCustomerID),
	}
	pmAttachParams.SetStripeAccount(p.AccountID)
	newPM, err := paymentmethod.Attach(pm.ID, pmAttachParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error attaching payment method to the connected account: %v", err)
	}

	// Update invoice settings, adding default payment method
	customerParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(newPM.ID),
		},
	}
	customerParams.SetStripeAccount(p.AccountID)
	cus, err := customer.Update(p.ConnectedCustomerID, customerParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error setting default payment method id for customer: %q", err)
	}

	return cus, nil
}

// ConfirmConnectPaymentIntent attempts to confirm a stripe connect payment intent with the customer default payment method
// The customer MUST have a default payment method, otherwise an error is returned
func (s *Service) ConfirmConnectPaymentIntent(p ConfirmConnectPaymentIntentParams) (*stripe.PaymentIntent, error) {
	// Retrieve up-to-date customer
	cusParams := &stripe.CustomerParams{}
	cusParams.SetStripeAccount(p.AccountID)
	cusParams.AddExpand("invoice_settings.default_payment_method")
	cus, err := customer.Get(p.CustomerID, cusParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "customer not found: %v", err)
	}
	if cus.InvoiceSettings.DefaultPaymentMethod == nil {
		return nil, status.Error(codes.Internal, "customer does not have a default pm")
	}

	// Confirm Payment Intent with customer default payment method
	params := &stripe.PaymentIntentConfirmParams{
		PaymentMethod: stripe.String(cus.InvoiceSettings.DefaultPaymentMethod.ID),
	}
	params.SetStripeAccount(p.AccountID)
	pm, err := paymentintent.Confirm(p.PaymentIntentID, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to confirm stripe payment intent: %v", err)
	}

	return pm, nil
}

// GetOrCreateConnectedCustomerByUserID tries to retrieve the connected customer on db. If it does not exist, a new one is created
func (s *Service) GetOrCreateConnectedCustomerByUserID(p GetOrCreateConnectedCustomerParams) (*v1API.ConnectedCustomer, error) {
	customersDir := customers.NewDirectory(s.db) // TODO: dont replicate that

	connectedCus, err := customersDir.GetConnectedCustomerByUserID(s.ctx, &customers.GetConnectedCustomerByUserIDParams{
		UserID:    p.UserID,
		AccountID: p.AccountID,
	})
	if err == nil {
		// Connected customer exists, no other action is required
		return connectedCus, nil
	}
	// Handle other db error
	if err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "failed to retrieve customer: %v", err)
	}

	// Connected customer does not exist. Create it
	connCus, err := s.CreateConnectedCustomer(&CreateConnectedCustomerParams{
		CustomerID: p.CustomerID,
		AccountID:  p.AccountID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create connected customer: %v", err)
	}
	// Store connected customer on db
	storedConnCus, err := customersDir.StoreConnectedCustomer(s.ctx, &customers.StoreConnectedCustomerParams{
		UserID:              p.UserID,
		CustomerID:          p.CustomerID,
		ConnectedCustomerID: connCus.ID,
		AccountID:           p.AccountID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error storing connected customer")
	}

	return storedConnCus, nil
}
