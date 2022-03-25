package v1_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DagDigg/unpaper/backend/customers"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/stretchr/testify/assert"
	stripe "github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/paymentmethod"
	"github.com/stripe/stripe-go/v72/sub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUpgradeSubscription(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When successfully signing up, creating and upgrading subscription", func(t *testing.T) {
		t.Parallel()
		createUserResp := createE2EUser(t, ws)
		// A customer must have been created,
		// alongside the free tier subscription
		customersDir := customers.NewDirectory(ws.Server.GetDB())

		// Assert customer
		cus, err := customersDir.GetCustomerByUserID(createUserResp.ctx, createUserResp.user.Id)
		assert.Nil(err)

		assert.Equal(len(cus.Subscriptions), 1)
		assert.Equal(cus.Subscriptions[0].Price.Id, ws.Cfg.PriceIDFree)
		// Free plan renews monthly, we dont have control on the stripe period end,
		// so we assert that the period end is between 26 and 32 days
		assert.Greater(cus.Subscriptions[0].CurrentPeriodEnd, time.Now().Add(24*time.Hour*26).Unix())
		assert.Less(cus.Subscriptions[0].CurrentPeriodEnd, time.Now().Add(24*time.Hour*32).Unix())
		assert.Equal(cus.Subscriptions[0].Status, v1API.SubscriptionStatus_ACTIVE)

		pm := createStripePM(t, testCardVisa)

		// Upgrade subscription
		ctx := metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs(
				"x-user-id", createUserResp.user.Id,
			),
		)
		upgradeReq := &v1API.SubscribeToPlanRequest{
			Plan:            v1API.Plan_UNPAPER_PLUS_MONTHLY,
			PaymentMethodId: pm.ID,
		}
		customerResp, err := ws.Server.SubscribeToPlan(ctx, upgradeReq)
		assert.Nil(err)
		assert.Equal(len(customerResp.Subscriptions), 1)

		// Assert subscription
		assert.Equal(customerResp.Subscriptions[0].CustomerId, cus.CustomerId)
		assert.Equal(customerResp.Subscriptions[0].Price.Plan, v1API.Plan_UNPAPER_PLUS_MONTHLY)
		assert.Equal(customerResp.Subscriptions[0].Price.Id, ws.Cfg.PriceIDPlusMonthly)

		// Assert invoice

		// Payment intent should not exist, as by default there's a trial period,
		// which doesn't generate a payment intent
		assert.Nil(customerResp.Subscriptions[0].LatestInvoice.PaymentIntent)
		assert.NotNil(customerResp.DefaultPaymentMethod)
		assert.Equal(customerResp.DefaultPaymentMethod.Id, pm.ID)
		assert.Equal(customerResp.DefaultPaymentMethod.LastFour, pm.Card.Last4)
		assert.Equal(customerResp.DefaultPaymentMethod.ExpMonth, int32(pm.Card.ExpMonth))
		assert.Equal(customerResp.DefaultPaymentMethod.ExpYear, int32(pm.Card.ExpYear))

		trialUsed, err := customersDir.HasCustomerUsedTrial(ctx, cus.CustomerId)
		assert.Nil(err)
		assert.True(trialUsed)

		// Upgrade to yearly
		upgradeReq = &v1API.SubscribeToPlanRequest{
			Plan:            v1API.Plan_UNPAPER_PLUS_YEARLY,
			PaymentMethodId: pm.ID,
		}
		customerResp, err = ws.Server.SubscribeToPlan(ctx, upgradeReq)
		assert.Nil(err)

		assert.Less(customerResp.Subscriptions[0].CurrentPeriodEnd, time.Now().Add(30*time.Second).Unix())

		// // Update customer payment method with an errored card
		// updateStripePM(t, customerResp.DefaultPaymentMethod.Id, testCard3DSInsufficientFunds)
		// // Update subscription to trigger trial end
		// subParams := &stripe.SubscriptionParams{
		// 	TrialEnd: stripe.Int64(time.Now().Unix()),
		// 	Items: []*stripe.SubscriptionItemsParams{
		// 		{
		// 			ID:    stripe.String(customerResp.Subscriptions[0].Items[0].Id),
		// 			Price: stripe.String(ws.Cfg.PriceIDPlusMonthly),
		// 		},
		// 	},
		// }
		// sub.Update(customerResp.Subscriptions[0].Id, subParams)
	})
}

func TestGetSubscriptionByID(t *testing.T) {
	t.Parallel()
	_ = v1Testing.GetWrappedServer(t)

	t.Run("When user subscribes and subscription is retrieved", func(t *testing.T) {

	})
}

func TestCreateAndConfirmSetupIntent(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When successfully creating a setup intent", func(t *testing.T) {
		t.Parallel()
		createReq := &v1API.CreateSetupIntentRequest{
			Api: "v1",
		}
		ctx := context.Background()
		createRes, err := ws.Server.CreateSetupIntent(ctx, createReq)
		assert.Nil(err)

		assert.NotEqual(createRes.Id, "")
		assert.NotEqual(createRes.ClientSecret, "")
	})
}

func TestInvoicePreview(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When user subscribes in free trial and gets invoice previews", func(t *testing.T) {
		userRes := createE2EUser(t, ws)

		cus, err := ws.Server.CustomerInfo(userRes.ctx, &v1API.CustomerInfoRequest{Api: "v1"})
		assert.Nilf(err, "%s", err)

		assert.Equal(len(cus.Subscriptions), 1)
		subs, err := ws.Server.GetSubscriptionByID(userRes.ctx, &v1API.GetSubscriptionByIDRequest{Api: "v1", SubscriptionId: cus.Subscriptions[0].Id})
		assert.Nil(err)
		assert.Equal(len(subs.Subscription.Items), 1)

		// Call invoice preview for plan Free => Plus Monthly
		inPreviewMonthly, err := ws.Server.InvoicePreview(metadata.NewIncomingContext(userRes.ctx, metadata.Pairs("x-user-id", userRes.user.Id)), &v1API.InvoicePreviewRequest{
			Api:                "v1",
			Plan:               v1API.Plan_UNPAPER_PLUS_MONTHLY,
			CustomerId:         cus.CustomerId,
			SubscriptionId:     subs.Subscription.Id,
			SubscriptionItemId: subs.Subscription.Items[0].Id,
		})
		assert.Nil(err)
		assert.Equal(inPreviewMonthly.AmountDue, int64(2999))

		// Call invoice preview for plan Free => Plus Yearly
		inPreviewYearly, err := ws.Server.InvoicePreview(userRes.ctx, &v1API.InvoicePreviewRequest{
			Api:                "v1",
			Plan:               v1API.Plan_UNPAPER_PLUS_YEARLY,
			CustomerId:         cus.CustomerId,
			SubscriptionId:     subs.Subscription.Id,
			SubscriptionItemId: subs.Subscription.Items[0].Id,
		})
		assert.Nil(err)
		assert.Equal(inPreviewYearly.AmountDue, int64(26990))
	})
}

type createE2EUserResp struct {
	user *v1API.User
	ctx  context.Context
}

func createE2EUser(t *testing.T, ws *v1Testing.WrappedServer) *createE2EUserResp {
	// Insert fresh user. When a request gets to the server,
	// external authorization already has stored user on db
	userParams := v1Testing.GetRandomPGUserParams()
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &v1Testing.ServerTransportStreamMock{
		SendHeaderFunc: func(md metadata.MD) error {
			return nil
		},
	})
	req := &v1API.EmailSignupRequest{Api: "v1", Email: userParams.Email, Password: userParams.Password.String, Username: userParams.Username.String}
	userRes, err := ws.Server.EmailSignup(ctx, req)
	if err != nil {
		t.Errorf("unexpected errror during rpc call: '%v'", err)
	}

	customersDir := customers.NewDirectory(ws.Server.GetDB())
	cus, err := customersDir.GetCustomerByUserID(ctx, userRes.Id)
	assert.Nil(t, err)
	t.Cleanup(func() {
		deleteStripeCustomer(t, cus.CustomerId)
	})

	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("x-user-id", userRes.Id))

	return &createE2EUserResp{
		user: userRes,
		ctx:  ctx,
	}
}

func createStripePM(t *testing.T, cardNumber testCard) *stripe.PaymentMethod {
	// Create payment method
	params := &stripe.PaymentMethodParams{
		Card: &stripe.PaymentMethodCardParams{
			Number:   stripe.String(string(cardNumber)),
			ExpMonth: stripe.String("3"),
			ExpYear:  stripe.String("2052"),
			CVC:      stripe.String("314"),
		},
		Type: stripe.String("card"),
	}
	pm, err := paymentmethod.New(params)
	if err != nil {
		t.Errorf("failed to create stripe payment method: %q", err)
	}

	return pm
}

func updateStripePM(t *testing.T, pmID string, cardNumber testCard) *stripe.PaymentMethod {
	// Update payment method
	params := &stripe.PaymentMethodParams{
		Card: &stripe.PaymentMethodCardParams{
			Number:   stripe.String(string(cardNumber)),
			ExpMonth: stripe.String("3"),
			ExpYear:  stripe.String("2022"),
			CVC:      stripe.String("314"),
		},
		Type: stripe.String("card"),
	}
	pm, err := paymentmethod.Update(pmID, params)
	if err != nil {
		t.Errorf("failed to update stripe payment method: %q", err)
	}

	return pm
}

func deleteStripeCustomer(t *testing.T, customerID string) {
	cus, err := customer.Del(customerID, nil)
	assert.Nil(t, err)
	_, err = sub.Cancel(cus.Subscriptions.Data[0].ID, nil)
	assert.Nil(t, err)
}

// Stripe Cards for testing
type testCard string

const (
	testCardVisa                 testCard = "4242424242424242"
	testCard3DSOnlyFirstPayment  testCard = "4000002500003155"
	testCard3DSAlways            testCard = "4000002760003184"
	testCard3DSInsufficientFunds testCard = "4000008260003178"
)

func TestRedis(t *testing.T) {
	ws := v1Testing.GetWrappedServer(t)
	fmt.Println(ws.Server.GetRDB().Ping(context.Background()))
}
