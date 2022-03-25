package v1

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/DagDigg/unpaper/backend/customers"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/mdutils"
	"github.com/DagDigg/unpaper/backend/pkg/stripeservice"
	"github.com/DagDigg/unpaper/backend/users"
	"github.com/Masterminds/squirrel"
	"github.com/golang/protobuf/ptypes/empty"
	stripe "github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/account"
	"github.com/stripe/stripe-go/v72/accountlink"
	"github.com/stripe/stripe-go/v72/coupon"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/invoice"
	"github.com/stripe/stripe-go/v72/loginlink"
	"github.com/stripe/stripe-go/v72/paymentmethod"
	"github.com/stripe/stripe-go/v72/setupintent"
	"github.com/stripe/stripe-go/v72/sub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SubscribeToPlan upgrades the stripe subscription to the request parameter plan.
// It also attempts to pay it, and on success, an email is sent to the customer
func (s *unpaperServiceServer) SubscribeToPlan(ctx context.Context, req *v1API.SubscribeToPlanRequest) (*v1API.Customer, error) {
	if req.PaymentMethodId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing payment method ID")
	}
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	customersDir := customers.NewDirectory(s.db)

	// Get customer by userID from database
	cus, err := customersDir.GetCustomerByUserID(ctx, userID)
	if err != nil && err != customers.ErrNoCustomer {
		return nil, status.Errorf(codes.Internal, "error retrieving customer: %q", err)
	}
	if err == customers.ErrNoCustomer {
		// There was a failure during customer creation
		// on user sign up. Recreate it
		usersDir := users.NewDirectory(s.db)

		user, err := usersDir.GetUser(ctx, userID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error getting user: %q", err)
		}

		stripesvc := stripeservice.New(ctx, s.db, s.cfg)
		stripeCus, err := stripesvc.CreateCustomer(&stripeservice.CreateCustomerParams{
			UserID:     user.Id,
			Email:      user.Email,
			FamilyName: user.FamilyName,
			GivenName:  user.GivenName,
		})

		if err != nil {
			return nil, err
		}
		_, err = stripesvc.StoreCustomer(stripeCus, stripeservice.StoreCustomerParams{
			UserID:     user.Id,
			FamilyName: user.FamilyName,
			GivenName:  user.GivenName,
		})
		if err != nil {
			return nil, err
		}
	}

	// Attach PaymentMethod to customer using the received payment method ID
	params := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(cus.CustomerId),
	}
	pm, err := paymentmethod.Attach(
		req.PaymentMethodId,
		params,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error attaching payment method to customer: %q", err)
	}

	// Update invoice settings, adding default payment method
	customerParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm.ID),
		},
	}
	customerParams.AddExpand("subscriptions")
	customerParams.AddExpand("invoice_settings.default_payment_method")
	stripeCustomer, err := customer.Update(cus.CustomerId, customerParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error updating customer: %q", err)
	}
	if stripeCustomer.Subscriptions == nil {
		return nil, status.Error(codes.Internal, "customer has no subscriptions. No update can be made")
	}

	// Update subscription
	stripesvc := stripeservice.New(ctx, s.db, s.cfg)
	subsc, err := stripesvc.UpdatePlatformSubscription(&stripeservice.UpdatePlatformSubscriptionParams{
		Plan:     req.Plan,
		Customer: stripeCustomer,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error updating subscription: %q", err)
	}

	// Store subscription on db
	pbSubsc, err := stripesvc.StorePlatformSubscription(&stripeservice.StorePlatformSubscriptionParams{
		Subscription: subsc,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error storing subscription: %v", err)
	}

	// Store payment method
	pbPM, err := customersDir.StoreDefaultPM(ctx, &customers.StoreDefaultPaymentMethodParams{
		ID:         pm.ID,
		UserID:     userID,
		CustomerID: pm.Customer.ID,
		LastFour:   pm.Card.Last4,
		ExpMonth:   int32(pm.Card.ExpMonth),
		ExpYear:    int32(pm.Card.ExpYear),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store payment method: %q", err)
	}

	return &v1API.Customer{
		Id:                   userID,
		CustomerId:           subsc.Customer.ID,
		Subscriptions:        []*v1API.Subscription{pbSubsc},
		DefaultPaymentMethod: pbPM,
	}, nil
}

// UpdateSubscription updates the customer's active subscription. It changes the period anchor
// if the update comes from the free plan to a paid one. It uses the attached customer payment method
// to perform the payment. If a customer has none, an error is returned
func (s *unpaperServiceServer) UpdateSubscription(ctx context.Context, req *v1API.UpdateSubscriptionRequest) (*v1API.Customer, error) {
	if req.CustomerId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing customer ID")
	}
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	// Retrieve up-to-date customer from stripe
	customerParams := &stripe.CustomerParams{}
	customerParams.AddExpand("subscriptions")
	customerParams.AddExpand("invoice_settings.default_payment_method")
	stripeCustomer, err := customer.Get(req.CustomerId, customerParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get customer from stripe: %q", err)
	}

	// Update stripe subscription
	stripesvc := stripeservice.New(ctx, s.db, s.cfg)
	subsc, err := stripesvc.UpdatePlatformSubscription(&stripeservice.UpdatePlatformSubscriptionParams{
		Plan:     req.Plan,
		Customer: stripeCustomer,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error updating subscription: %q", err)
	}

	// Store subscription on db
	pbSubsc, err := stripesvc.StorePlatformSubscription(&stripeservice.StorePlatformSubscriptionParams{
		Subscription: subsc,
	})

	defaultPM, err := stripePaymentMethodToPB(stripeCustomer.InvoiceSettings.DefaultPaymentMethod)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error converting stripe default payment method to protobuf: %q", err)
	}
	return &v1API.Customer{
		Id:                   userID,
		CustomerId:           subsc.Customer.ID,
		Subscriptions:        []*v1API.Subscription{pbSubsc},
		DefaultPaymentMethod: defaultPM,
	}, nil
}

// CustomerInfo returns the v1API.Customer, with its subscriptions attached
func (s *unpaperServiceServer) CustomerInfo(ctx context.Context, req *v1API.CustomerInfoRequest) (*v1API.Customer, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	customersDir := customers.NewDirectory(s.db)

	pbCustomer, err := customersDir.GetCustomerByUserID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error retrieving customer by user id: %q", err)
	}
	pbSub, err := customersDir.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		if err != sql.ErrNoRows {
			// It's ok if customer has no subscriptions
			return nil, status.Errorf(codes.Internal, "error retrieving subscription by user id: %q", err)
		}
	}
	if pbSub != nil {
		// At the moment only one subscription at time
		pbCustomer.Subscriptions = []*v1API.Subscription{pbSub}
	}
	return pbCustomer, nil
}

// RetryInvoice updates the customer with the new payment method,
// and assigns it as the new default payment method for subscription invoices.
func (s *unpaperServiceServer) RetryInvoice(ctx context.Context, req *v1API.RetryInvoiceRequest) (*v1API.Invoice, error) {
	if req.CustomerId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing customer id")
	}
	if req.InvoiceId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing invoice id")
	}
	if req.PaymentMethodId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing payment method id")
	}

	// Attach PaymentMethod
	params := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(req.CustomerId),
	}
	pm, err := paymentmethod.Attach(
		req.PaymentMethodId,
		params,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error occurred attaching payment method to customer: %q", err)
	}

	// Update invoice settings default
	customerParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm.ID),
		},
	}
	_, err = customer.Update(
		req.CustomerId,
		customerParams,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error occurred updating customer: %q", err)
	}

	// Retrieve Invoice
	invoiceParams := &stripe.InvoiceParams{}
	invoiceParams.AddExpand("payment_intent")
	in, err := invoice.Get(req.InvoiceId, invoiceParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve invoice: %v", err)
	}
	piStatus, err := stripeservice.StripePaymentIntentStatusToPB(in.PaymentIntent.Status)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error converting payment intent status: %q", err)
	}
	return &v1API.Invoice{
		Id:        in.ID,
		Subtotal:  in.Subtotal,
		AmountDue: in.AmountDue,
		PaymentIntent: &v1API.PaymentIntent{
			Status: piStatus,
		},
	}, nil
}

// GetSubscriptionByID retrieves the subscription with the provided ID
// directly from stripe, and not from the database. And then the protobuf version is returned
func (s *unpaperServiceServer) GetSubscriptionByID(ctx context.Context, req *v1API.GetSubscriptionByIDRequest) (*v1API.GetSubscriptionByIDResponse, error) {
	if req.SubscriptionId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing subscription id")
	}

	params := &stripe.SubscriptionParams{}
	params.AddExpand("latest_invoice.payment_intent")
	stripeSubs, err := sub.Get(req.SubscriptionId, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve subscription from stripe: %q", err)
	}

	pbSubsc, err := stripeservice.StripeSubscriptionToPB(s.cfg, stripeservice.StripeSubscriptionToPBParams{
		Subscription: stripeSubs,
		Options: &stripeservice.StripeSubscriptionOptions{
			WithLatestInvoice: true,
			WithItems:         true,
			WithPlan:          true,
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert stripe subscription to PB: %q", err)
	}
	return &v1API.GetSubscriptionByIDResponse{
		Subscription: pbSubsc,
	}, nil
}

// CreateSetupIntent creates a Stripe setup intent and returns its ID.
// From Stripe: "A SetupIntent guides you through the process of
// setting up and saving a customer's payment credentials for future payments."
func (s *unpaperServiceServer) CreateSetupIntent(ctx context.Context, req *v1API.CreateSetupIntentRequest) (*v1API.CreateSetupIntentResponse, error) {
	params := &stripe.SetupIntentParams{
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
	}
	si, err := setupintent.New(params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create stripe setup intent: %q", err)
	}

	return &v1API.CreateSetupIntentResponse{
		Id:           si.ID,
		ClientSecret: si.ClientSecret,
	}, nil
}

func (s *unpaperServiceServer) AttachPaymentMethod(ctx context.Context, req *v1API.AttachPaymentMethodRequest) (*v1API.PaymentMethod, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	if req.PaymentMethodId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing payment method id")
	}
	if req.CustomerId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing customer id")
	}

	// List customer payment methods
	pmListParams := &stripe.PaymentMethodListParams{
		Customer: stripe.String(req.CustomerId),
		Type:     stripe.String("card"),
	}
	l := paymentmethod.List(pmListParams)

	// Detach all payment methods
	for l.Next() {
		_, err := paymentmethod.Detach(l.PaymentMethod().ID, nil)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error detaching payment method from customer: %q", err)
		}
	}

	// Attach PaymentMethod to customer using the received payment method ID
	pmParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(req.CustomerId),
	}
	pm, err := paymentmethod.Attach(
		req.PaymentMethodId,
		pmParams,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error attaching payment method to customer: %q", err)
	}

	// Update invoice settings, adding default payment method
	customerParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm.ID),
		},
	}
	_, err = customer.Update(req.CustomerId, customerParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error setting default payment method id for customer: %q", err)
	}

	cusDir := customers.NewDirectory(s.db)

	// Delete all customer payment methods on db
	err = cusDir.ClearCustomerPaymentMethods(ctx, pm.Customer.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error deleting customers payment methods: %q", err)
	}

	// Store new payment method
	pmPB, err := cusDir.StoreDefaultPM(ctx, &customers.StoreDefaultPaymentMethodParams{
		ID:         pm.ID,
		UserID:     userID,
		CustomerID: pm.Customer.ID,
		LastFour:   pm.Card.Last4,
		ExpMonth:   int32(pm.Card.ExpMonth),
		ExpYear:    int32(pm.Card.ExpYear),
	})
	if err != nil {
		// Failure storing payment method.
		return nil, status.Errorf(codes.Internal, "failed to store payment method %v", err)
	}

	return pmPB, nil
}

// InvoicePreview calculates an invoice based on the customer id and plan.
// It returns the invoice id and amount
func (s *unpaperServiceServer) InvoicePreview(ctx context.Context, req *v1API.InvoicePreviewRequest) (*v1API.Invoice, error) {
	if req.CustomerId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing customer id")
	}
	if req.SubscriptionId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing subscription id")
	}
	if req.SubscriptionItemId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing subscription item id")
	}

	stripesvc := stripeservice.New(ctx, s.db, s.cfg)
	pbInvoice, err := stripesvc.GetPlatformInvoicePreview(&stripeservice.GetPlatformInvoicePreviewParams{
		CustomerID:         req.CustomerId,
		SubscriptionID:     req.SubscriptionId,
		SubscriptionItemID: req.SubscriptionItemId,
		Plan:               req.Plan,
		Coupon:             req.Coupon,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve invoice preview: %q", err)
	}

	return pbInvoice, nil
}

// CouponCheck calls Stripe to check the validity of the coupon
func (s *unpaperServiceServer) CouponCheck(ctx context.Context, req *v1API.CouponCheckRequest) (*v1API.CouponCheckResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "missing coupon id")
	}

	c, err := coupon.Get(req.Id, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error retrieving coupon: %v", err)
	}
	if !c.Valid {
		return nil, status.Error(codes.Internal, "invalid coupon")
	}

	return &v1API.CouponCheckResponse{
		Valid: true,
	}, nil
}

func createStripeStandardAccount() (*stripe.Account, error) {
	params := &stripe.AccountParams{
		Type: stripe.String(string(stripe.AccountTypeStandard)),
		Capabilities: &stripe.AccountCapabilitiesParams{
			CardPayments: &stripe.AccountCapabilitiesCardPaymentsParams{
				Requested: stripe.Bool(true),
			},
		},
	}

	return account.New(params)
}

// ErrNilPaymentMethod indicates the absence of a *stripe.PaymentMethod
var ErrNilPaymentMethod = fmt.Errorf("stripe payment method is nil")

// stripePaymentMethodToPB converts stripe payment method to PB. No user ID is attached
func stripePaymentMethodToPB(pm *stripe.PaymentMethod) (*v1API.PaymentMethod, error) {
	if pm == nil {
		return nil, ErrNilPaymentMethod
	}
	return &v1API.PaymentMethod{
		Id:        pm.ID,
		LastFour:  pm.Card.Last4,
		ExpMonth:  int32(pm.Card.ExpMonth),
		ExpYear:   int32(pm.Card.ExpYear),
		IsDefault: true,
	}, nil
}

// GetConnectAccountLink generates a stripe connect link.
// The account id is retrieved by the incoming user id in the request
func (s *unpaperServiceServer) GetConnectAccountLink(ctx context.Context, req *empty.Empty) (*v1API.GetConnectAccountLinkResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	var acctID string
	q := s.GetSB().Select("account_id").From("customers").Where(squirrel.Eq{"id": userID})
	err := q.QueryRowContext(ctx).Scan(&acctID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error scanning account id: %v", err)
	}

	params := &stripe.AccountLinkParams{
		Account:    stripe.String(acctID),
		RefreshURL: stripe.String("https://localhost:3000/refresh-connect-link"),
		ReturnURL:  stripe.String("https://localhost:3000"),
		Type:       stripe.String("account_onboarding"),
	}
	acc, err := accountlink.New(params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create account link: %v", err)
	}

	return &v1API.GetConnectAccountLinkResponse{
		Link: acc.URL,
	}, nil
}

// MakeDonation creates a payment intent on behalf of a connected account id, and attempts to confirm it
// using the connected customer default payment_method
func (s *unpaperServiceServer) MakeDonation(ctx context.Context, req *v1API.MakeDonationRequest) (*v1API.ConnectedPaymentIntentResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	if req.Amount == 0 {
		return nil, status.Error(codes.InvalidArgument, "amount cannot be zero")
	}
	if req.ReceiverUserId == "" {
		return nil, status.Error(codes.InvalidArgument, "receiver user id cannot be empty")
	}

	customersDir := customers.NewDirectory(s.db)

	// Check if connected account exists and can receive donations
	connectedAcct, err := customersDir.GetConnectedAccountByUserID(ctx, req.ReceiverUserId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.Internal, "connected account not found: %v", err)
		}
	}
	if !connectedAcct.CanReceivePayments {
		return nil, status.Error(codes.Internal, "connected account cannot receive donations")
	}

	feePct, err := strconv.ParseFloat(s.cfg.ApplicationFeePercent, 64)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert env application fee amount to int: %v", err)
	}
	connectedCus, err := customersDir.GetConnectedCustomerByUserID(ctx, &customers.GetConnectedCustomerByUserIDParams{
		UserID:    userID,
		AccountID: connectedAcct.AccountId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "an error occurred: %v", err)
	}

	// Create payment intent
	stripesvc := stripeservice.New(ctx, s.db, s.cfg)
	piCreated, err := stripesvc.CreatePaymentIntent(&stripeservice.CreateConnectPaymentIntentParams{
		SenderConnectCustomerID: connectedCus.ConnectedCustomerId,
		ReceiverAccountID:       connectedAcct.AccountId,
		PlatformFeePercent:      feePct,
		Amount:                  int64(req.Amount),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create payment intent: %v", err)
	}

	// Confirm payment intent
	piConfirmed, err := stripesvc.ConfirmConnectPaymentIntent(stripeservice.ConfirmConnectPaymentIntentParams{
		PaymentIntentID: piCreated.ID,
		AccountID:       connectedAcct.AccountId,
		CustomerID:      connectedCus.ConnectedCustomerId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to confirm payment intent: %v", err)
	}
	piStatus, err := stripeservice.StripePaymentIntentStatusToPB(piConfirmed.Status)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert stripe PI status to pb: %v", err)
	}

	return &v1API.ConnectedPaymentIntentResponse{
		PaymentIntent: &v1API.PaymentIntent{
			Id:           piConfirmed.ID,
			Status:       piStatus,
			ClientSecret: piConfirmed.ClientSecret,
		},
		AccountId: connectedAcct.AccountId,
	}, nil
}

// PayRoomEntrance creates a payment intent for accessing the paid room and attempts to pay it
// using the connected customer default payment method id
func (s *unpaperServiceServer) PayRoomEntrance(ctx context.Context, req *v1API.PayRoomEntranceRequest) (*v1API.ConnectedPaymentIntentResponse, error) {
	// userID, ok := mdutils.GetUserIDFromMD(ctx)
	// if !ok {
	// 	return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	// }
	// if req.RoomId == "" {
	// 	return nil, status.Error(codes.InvalidArgument, "room id cannot be zero")
	// }

	// roomsDir := rooms.NewDirectory(s.db)
	// customersDir := customers.NewDirectory(s.db)
	// stripesvc := stripeservice.New(ctx, s.db, s.cfg)

	// // Fetch room and check if it can be paid
	// roomData, err := getMustPaidRoomData(roomDataParams{
	// 	ctx:          ctx,
	// 	roomID:       req.RoomId,
	// 	customersDir: customersDir,
	// 	roomsDir:     roomsDir,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// platformSndCus, err := customersDir.GetCustomerByUserID(ctx, userID)
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to retrieve platform customer: %v", err)
	// }
	// // Retrieve connected customer, if it doesn't exist, create one
	// cus, err := stripesvc.GetOrCreateConnectedCustomerByUserID(stripeservice.GetOrCreateConnectedCustomerParams{
	// 	UserID:     userID,
	// 	CustomerID: platformSndCus.CustomerId,
	// 	AccountID:  roomData.owner.AccountId,
	// })

	// feePct, err := strconv.ParseFloat(s.cfg.ApplicationFeePercent, 64)
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to convert env application fee amount to int: %v", err)
	// }
	// pi, err := stripesvc.CreatePaymentIntent(&stripeservice.CreateConnectPaymentIntentParams{
	// 	SenderConnectCustomerID: cus.ConnectedCustomerId,
	// 	ReceiverAccountID:       roomData.owner.AccountId,
	// 	PlatformFeePercent:      feePct,
	// 	Amount:                  roomData.room.Price,
	// 	Metadata: map[string]string{
	// 		stripeservice.StripeMDPayForRoomAccess: roomData.room.Id,
	// 		stripeservice.StripeMDUserID:           userID,
	// 	},
	// })
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to create payment intent: %v", err)
	// }
	// confirmPI, err := stripesvc.ConfirmConnectPaymentIntent(stripeservice.ConfirmConnectPaymentIntentParams{
	// 	PaymentIntentID: pi.ID,
	// 	AccountID:       roomData.owner.AccountId,
	// 	CustomerID:      cus.ConnectedCustomerId,
	// })
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to confirm payment intent: %v", err)
	// }
	// piPB, err := stripeservice.StripePaymentIntentToPB(confirmPI)
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to convert stripe PI to PB: %v", err)
	// }

	// return &v1API.ConnectedPaymentIntentResponse{
	// 	PaymentIntent: piPB,
	// 	AccountId:     roomData.owner.AccountId,
	// }, nil
	return nil, nil
}

// CreateStripeAccount creates an express account and links it to the customers table based on the incoming userID
func (s *unpaperServiceServer) CreateStripeAccount(ctx context.Context, req *empty.Empty) (*v1API.Customer, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	acct, err := createStripeStandardAccount()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error creating stripe account: %q", err)
	}

	customersDir := customers.NewDirectory(s.db)
	cus, err := customersDir.SetAccountIDFromUserID(ctx, &customers.SetAccountIDFromUserIDParams{
		AccountID: sql.NullString{String: acct.ID, Valid: true},
		ID:        userID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update customer account id")
	}

	return cus, nil
}

// GetDashboardLink generetes a link for viewing the stripe dashboard.
// The user must have a connected stripe account
func (s *unpaperServiceServer) GetDashboardLink(ctx context.Context, req *empty.Empty) (*v1API.GetDashboardLinkResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	customersDir := customers.NewDirectory(s.db)

	acctID, err := customersDir.GetAccountIDFromUserID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "customer account_id not found: %v", err)
	}

	params := &stripe.LoginLinkParams{
		Account: stripe.String(acctID),
	}
	link, err := loginlink.New(params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate login link: %v", err)
	}

	return &v1API.GetDashboardLinkResponse{
		Link: link.URL,
	}, nil
}

// CheckRoomEntrancePI checks if the customer has a payment intent
// associated with the room id, if so, it returns its status
func (s *unpaperServiceServer) CheckRoomEntrancePI(ctx context.Context, req *v1API.CheckRoomEntrancePIRequest) (*v1API.CheckRoomEntrancePIResponse, error) {
	// userID, ok := mdutils.GetUserIDFromMD(ctx)
	// if !ok {
	// 	return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	// }
	// if req.RoomId == "" {
	// 	return nil, status.Error(codes.InvalidArgument, "missing room id")
	// }

	// customersDir := customers.NewDirectory(s.db)
	// roomsDir := rooms.NewDirectory(s.db)

	// roomData, err := getMustPaidRoomData(roomDataParams{
	// 	ctx:          ctx,
	// 	roomID:       req.RoomId,
	// 	customersDir: customersDir,
	// 	roomsDir:     roomsDir,
	// })
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "error getting room data: %v", err)
	// }

	// cus, err := customersDir.GetConnectedCustomerByUserID(ctx, &customers.GetConnectedCustomerByUserIDParams{
	// 	UserID:    userID,
	// 	AccountID: roomData.owner.AccountId,
	// })
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to get connected customer by user id: %v", err)
	// }

	// params := &stripe.PaymentIntentListParams{
	// 	Customer: stripe.String(cus.CustomerId),
	// }
	// params.SetStripeAccount(roomData.owner.AccountId)
	// i := paymentintent.List(params)
	// for i.Next() {
	// 	pi := i.PaymentIntent()
	// 	roomID, ok := pi.Metadata[stripeservice.StripeMDPayForRoomAccess]
	// 	if ok && roomID == req.RoomId {
	// 		// Found payment intent for room
	// 		piStatus, err := stripeservice.StripePaymentIntentStatusToPB(pi.Status)
	// 		if err != nil {
	// 			return nil, status.Errorf(codes.Internal, "failed to convert stripe pi status to pb: %v", err)
	// 		}
	// 		return &v1API.CheckRoomEntrancePIResponse{
	// 			PiFound:  true,
	// 			PiStatus: piStatus,
	// 		}, nil
	// 	}
	// }

	// return &v1API.CheckRoomEntrancePIResponse{
	// 	PiFound: false,
	// }, nil
	return nil, nil
}

// SubscribeToRoom creates a subscription for that specific room and attempts to pay it with the default customer PM.
// It must be called for NEW subscriptions. If the customer has already subscribed and the subscription is in an error state,
// a retry on the subscription is required, but not a new subscription. Any attempt to re create a subscription will be rejected
func (s *unpaperServiceServer) SubscribeToRoom(ctx context.Context, req *v1API.SubscribeToRoomRequest) (*v1API.SubscribeToRoomResponse, error) {
	// userID, ok := mdutils.GetUserIDFromMD(ctx)
	// if !ok {
	// 	return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	// }
	// if req.RoomId == "" {
	// 	return nil, status.Error(codes.InvalidArgument, "missing room id")
	// }

	// stripesvc := stripeservice.New(ctx, s.db, s.cfg)
	// customersDir := customers.NewDirectory(s.db)
	// roomsDir := rooms.NewDirectory(s.db)

	// // Fetch room data
	// roomData, err := getMustPaidRoomData(roomDataParams{
	// 	ctx:          ctx,
	// 	roomID:       req.RoomId,
	// 	customersDir: customersDir,
	// 	roomsDir:     roomsDir,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// platformSndCus, err := customersDir.GetCustomerByUserID(ctx, userID)
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to retrieve platform customer: %v", err)
	// }
	// // Retrieve connected customer, if it doesn't exist, create one
	// cus, err := stripesvc.GetOrCreateConnectedCustomerByUserID(stripeservice.GetOrCreateConnectedCustomerParams{
	// 	UserID:     userID,
	// 	CustomerID: platformSndCus.CustomerId,
	// 	AccountID:  roomData.owner.AccountId,
	// })

	// // Before subscribing, there needs to be a check on the customer.
	// // The customer must not have a room subscription
	// hasAlreadySubscribed := isSubscribedToRoom(cus.ConnectedCustomerId, roomData.owner.AccountId, req.RoomId)
	// if hasAlreadySubscribed {
	// 	return nil, status.Errorf(codes.FailedPrecondition, "customer already has a subscription on the room")
	// }

	// // Calculate fees
	// baseFeePct, err := strconv.ParseFloat(s.cfg.ApplicationFeePercent, 64)
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to convert application fee amount: %v", err)
	// }

	// // Subscribe
	// stripesubsc, err := stripesvc.CreateRoomSubscription(&stripeservice.CreateRoomSubscriptionParams{
	// 	SenderConnectedCustomerID: cus.ConnectedCustomerId,
	// 	SenderUserID:              cus.UserId,
	// 	ReceiverAccountID:         roomData.owner.AccountId,
	// 	PlatformFeePct:            baseFeePct,
	// 	Amount:                    roomData.room.Price,
	// 	ProductID:                 roomData.room.ProductId,
	// 	Metadata: map[string]string{
	// 		stripeservice.StripeMDPayForRoomAccess: req.RoomId,
	// 		stripeservice.StripeMDUserID:           userID,
	// 	},
	// })
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to create subscription: %v", err)
	// }

	// pbSubsc, err := stripesvc.StoreRoomSubscription(&stripeservice.StoreRoomSubscriptionParams{
	// 	Subscription: stripesubsc,
	// 	RoomID:       req.RoomId,
	// 	AccountID:    roomData.owner.AccountId,
	// })
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to convert subscription to PB: %v", err)
	// }

	// return &v1API.SubscribeToRoomResponse{
	// 	Subscription: pbSubsc,
	// 	AccountId:    roomData.owner.AccountId,
	// }, nil
	return nil, nil
}

// isSubscribedToRoom returns whether a customer has a subscription
// on the provided roomID
func isSubscribedToRoom(connectedCusID, accountID, roomID string) bool {
	params := &stripe.SubscriptionListParams{Customer: connectedCusID}
	params.SetStripeAccount(accountID)
	i := sub.List(params)
	for i.Next() {
		s := i.Subscription()
		subRoomID, ok := s.Metadata[stripeservice.StripeMDPayForRoomAccess]
		if ok {
			if subRoomID == roomID {
				return true
			}
		}
	}

	return false
}

// GetRoomSubscriptions returns every room subscription associated with the incoming userID in metadata
func (s *unpaperServiceServer) GetRoomSubscriptions(ctx context.Context, req *empty.Empty) (*v1API.GetRoomSubscriptionsResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	customersDir := customers.NewDirectory(s.db)
	subsc, err := customersDir.GetRoomSubscriptionsForUserID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get room subscriptions: %v", err)
	}

	return &v1API.GetRoomSubscriptionsResponse{
		RoomSubscriptions: subsc,
	}, nil
}

// ConfirmRoomSubscription searches for a subscription of the provided user id and room id, fetches the most recent one,
// and tries to confirm the payment intent attached to its latest invoice. It should be called after updating the user PM
func (s *unpaperServiceServer) ConfirmRoomSubscription(ctx context.Context, req *v1API.ConfirmRoomSubscriptionRequest) (*v1API.ConfirmRoomSubscriptionResponse, error) {
	// userID, ok := mdutils.GetUserIDFromMD(ctx)
	// if !ok {
	// 	return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	// }
	// if req.RoomId == "" {
	// 	return nil, status.Error(codes.InvalidArgument, "missing room id")
	// }

	// customersDir := customers.NewDirectory(s.db)
	// roomsDir := rooms.NewDirectory(s.db)

	// // Fetch room data
	// roomData, err := getMustPaidRoomData(roomDataParams{
	// 	ctx:          ctx,
	// 	roomID:       req.RoomId,
	// 	customersDir: customersDir,
	// 	roomsDir:     roomsDir,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// // Get up-to-date subscription
	// stripesvc := stripeservice.New(ctx, s.db, s.cfg)
	// stripeSubsc, err := stripesvc.GetRoomSubscriptionForUserID(stripeservice.GetRoomSubscriptionForUserIDParams{
	// 	RoomID:    req.RoomId,
	// 	UserID:    userID,
	// 	AccountID: roomData.owner.AccountId,
	// 	Options: stripeservice.GetRoomSubscriptionForUserIDOptions{
	// 		StrictPaymentIntentCheck: true, // We want to strictly get latest invoice payment intent
	// 	},
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// // Confirm payment intent
	// _, err = stripesvc.ConfirmConnectPaymentIntent(stripeservice.ConfirmConnectPaymentIntentParams{
	// 	PaymentIntentID: stripeSubsc.LatestInvoice.PaymentIntent.ID,
	// 	AccountID:       roomData.owner.AccountId,
	// 	CustomerID:      stripeSubsc.LatestInvoice.Customer.ID,
	// })
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to confirm room payment intent: %v", err)
	// }

	// // Store subscription
	// subsc, err := stripesvc.StoreRoomSubscription(&stripeservice.StoreRoomSubscriptionParams{
	// 	Subscription: stripeSubsc,
	// 	RoomID:       req.RoomId,
	// 	AccountID:    roomData.owner.AccountId,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// return &v1API.ConfirmRoomSubscriptionResponse{
	// 	Subscription: subsc,
	// }, nil
	return nil, nil
}

// RetryRoomSubscriptions retrieve the room subscription by id, and attempts to confirming its latest invoice payment intent
// using the connected account default payment method
func (s *unpaperServiceServer) RetryRoomSubscription(ctx context.Context, req *v1API.RetryRoomSubscriptionRequest) (*v1API.ConnectedPaymentIntentResponse, error) {
	// userID, ok := mdutils.GetUserIDFromMD(ctx)
	// if !ok {
	// 	return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	// }
	// if req.RoomId == "" {
	// 	return nil, status.Error(codes.InvalidArgument, "missing room id")
	// }

	// roomsDir := rooms.NewDirectory(s.db)
	// customersDir := customers.NewDirectory(s.db)

	// // Fetch room data
	// roomData, err := getMustPaidRoomData(roomDataParams{
	// 	ctx:          ctx,
	// 	roomID:       req.RoomId,
	// 	customersDir: customersDir,
	// 	roomsDir:     roomsDir,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// // Get up-to-date subscription
	// stripesvc := stripeservice.New(ctx, s.db, s.cfg)
	// subsc, err := stripesvc.GetRoomSubscriptionForUserID(stripeservice.GetRoomSubscriptionForUserIDParams{
	// 	RoomID:    req.RoomId,
	// 	UserID:    userID,
	// 	AccountID: roomData.owner.AccountId,
	// 	Options: stripeservice.GetRoomSubscriptionForUserIDOptions{
	// 		StrictPaymentIntentCheck: true, // We want to strictly get latest invoice payment intent
	// 	},
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// // Store subscription
	// _, err = stripesvc.StoreRoomSubscription(&stripeservice.StoreRoomSubscriptionParams{
	// 	Subscription: subsc,
	// 	RoomID:       req.RoomId,
	// 	AccountID:    roomData.owner.AccountId,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// if subsc.LatestInvoice.PaymentIntent.Status != stripe.PaymentIntentStatusRequiresAction && subsc.LatestInvoice.PaymentIntent.Status != stripe.PaymentIntentStatusRequiresPaymentMethod {
	// 	return nil, status.Errorf(codes.Internal, "latest invoice payment intent has a status that prevents it to be retried: %q", subsc.LatestInvoice.PaymentIntent.Status)
	// }

	// pi, err := stripesvc.ConfirmConnectPaymentIntent(stripeservice.ConfirmConnectPaymentIntentParams{
	// 	PaymentIntentID: subsc.LatestInvoice.PaymentIntent.ID,
	// 	AccountID:       roomData.owner.AccountId,
	// 	CustomerID:      subsc.Customer.ID,
	// })
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to confirm payment intent: %v", err)
	// }
	// piPB, err := stripeservice.StripePaymentIntentToPB(pi)
	// if err != nil {
	// 	return nil, err
	// }

	// return &v1API.ConnectedPaymentIntentResponse{
	// 	PaymentIntent: piPB,
	// 	AccountId:     roomData.owner.AccountId,
	// }, nil
	return nil, nil
}

func (s *unpaperServiceServer) GetRoomSubscriptionByRoomID(ctx context.Context, req *v1API.GetRoomSubscriptionByRoomIDRequest) (*v1API.GetRoomSubscriptionByRoomIDResponse, error) {
	// userID, ok := mdutils.GetUserIDFromMD(ctx)
	// if !ok {
	// 	return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	// }
	// if req.RoomId == "" {
	// 	return nil, status.Error(codes.InvalidArgument, "missing room id")
	// }

	// roomsDir := rooms.NewDirectory(s.db)
	// customersDir := customers.NewDirectory(s.db)

	// // Fetch room data
	// roomData, err := getMustPaidRoomData(roomDataParams{
	// 	ctx:          ctx,
	// 	roomID:       req.RoomId,
	// 	customersDir: customersDir,
	// 	roomsDir:     roomsDir,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// // Get up-to-date subscription and store it
	// stripesvc := stripeservice.New(ctx, s.db, s.cfg)
	// subsc, err := stripesvc.RetrieveAndStoreRoomSubscriptionForUserID(stripeservice.GetRoomSubscriptionForUserIDParams{
	// 	RoomID:    req.RoomId,
	// 	UserID:    userID,
	// 	AccountID: roomData.owner.AccountId,
	// 	Options: stripeservice.GetRoomSubscriptionForUserIDOptions{
	// 		StrictPaymentIntentCheck: true, // We want to strictly get latest invoice payment intent
	// 	},
	// })
	// if err != nil {
	// 	if err == sql.ErrNoRows {
	// 		return nil, status.Errorf(codes.FailedPrecondition, "subscription does not exist")
	// 	}
	// 	return nil, err
	// }

	// return &v1API.GetRoomSubscriptionByRoomIDResponse{
	// 	Subscription: subsc,
	// }, nil
	return nil, nil
}

// GetOwnConnectedAccount retrieves and returns a users own connected account with restrictec fields. It is not the same as the protobuf `ConnectedAccount`
func (s *unpaperServiceServer) GetOwnConnectedAccount(ctx context.Context, req *empty.Empty) (*v1API.GetOwnConnectedAccountResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	customersDir := customers.NewDirectory(s.db)

	connectedAcct, err := customersDir.GetConnectedAccountByUserID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error retrieving connected account: %v", err)
	}

	return &v1API.GetOwnConnectedAccountResponse{
		CanReceivePayments: connectedAcct.CanReceivePayments,
	}, nil
}

type roomData struct {
	room  *v1API.Room
	owner *v1API.ConnectedAccount
}

type roomDataParams struct {
	ctx          context.Context
	roomID       string
	customersDir *customers.Directory
}

// getMustPaidRoomData calls `getRoomData` but makes checks on the room.
// The room returned must have a positive price, correct paid type, and the owner must be able to receive payments
func getMustPaidRoomData(p roomDataParams) (*roomData, error) {
	// roomData, err := getRoomData(roomDataParams{
	// 	ctx:          p.ctx,
	// 	roomID:       p.roomID,
	// 	customersDir: p.customersDir,
	// 	roomsDir:     p.roomsDir,
	// })
	// if err != nil {
	// 	return nil, err
	// }
	// if roomData.room.Price == 0 {
	// 	return nil, status.Error(codes.Internal, "room is free")
	// }
	// acceptedRoomTypes := []v1API.RoomType_Enum{v1API.RoomType_SUBSCRIPTION_MONTHLY, v1API.RoomType_PAID}

	// if !roomTypeContains(acceptedRoomTypes, roomData.room.RoomType) {
	// 	return nil, status.Errorf(codes.Internal, "unexpected room type: %v, only one-time pay and subsciptions are accepted", roomData.room.RoomType.String())
	// }

	// // Check if receiver customer can accept payments
	// if !roomData.owner.CanReceivePayments {
	// 	return nil, status.Error(codes.Internal, "connected account cant receive payments")
	// }

	// return roomData, err
	return nil, nil
}

// getRoomData retrieves the room and its owner by room_id
func getRoomData(p roomDataParams) (*roomData, error) {
	// Fetch room and check if it can be paid
	// room, err := p.roomsDir.GetRoomByID(p.ctx, p.roomID)
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to retrieve room: %v", err)
	// }

	// // Check if customer can receive donations
	// connectedAcct, err := p.customersDir.GetConnectedAccountByUserID(p.ctx, room.Owner)
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "error retrieving connected account: %v", err)
	// }

	// return &roomData{
	// 	room:  room,
	// 	owner: connectedAcct,
	// }, nil
	return nil, nil
}

func roomTypeContains(s []v1API.RoomType_Enum, v v1API.RoomType_Enum) bool {
	for _, t := range s {
		if t == v {
			return true
		}
	}

	return false
}
