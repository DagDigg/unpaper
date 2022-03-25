package customers_test

import (
	"context"
	"testing"
	"time"

	"github.com/DagDigg/unpaper/backend/customers"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCustomer(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	dir := getCustomersDirectory(t)

	t.Cleanup(func() {
		err := dir.Close()
		assert.Nil(err)
	})

	t.Run("When successfully adding and retrieving a customer with no subscriptions", func(t *testing.T) {
		t.Parallel()

		params := &customers.CreateCustomerParams{
			ID:         "myID",
			CustomerID: "myCustomerID",
			FirstName:  "Jane",
			LastName:   "Doe",
		}
		cus, err := dir.CreateCustomer(ctx, params)
		assert.Nil(err)

		assert.Equal(cus.Id, params.ID)
		assert.Equal(cus.FirstName, params.FirstName)
		assert.Equal(cus.LastName, params.LastName)

		// Retrieve customer
		retrievedCus, err := dir.GetCustomerByUserID(ctx, params.ID)
		assert.Nil(err)
		assert.Equal(retrievedCus.CustomerId, params.CustomerID)
		assert.Equal(retrievedCus.FirstName, params.FirstName)
		assert.Equal(retrievedCus.LastName, params.LastName)
		assert.Equal(retrievedCus.Id, params.ID)
	})

	t.Run("When a customer with no userID is added", func(t *testing.T) {
		t.Parallel()

		params := &customers.CreateCustomerParams{
			CustomerID: "myCustomerID",
			FirstName:  "Jane",
			LastName:   "Doe",
		}

		_, err := dir.CreateCustomer(ctx, params)
		assert.NotNil(err)
		assert.Equal(err, customers.ErrMissingUserID)
	})

	t.Run("When a customer with no customerID is added", func(t *testing.T) {
		t.Parallel()

		params := &customers.CreateCustomerParams{
			ID:        "myID",
			FirstName: "Jane",
			LastName:  "Doe",
		}

		_, err := dir.CreateCustomer(ctx, params)
		assert.NotNil(err)
		assert.Equal(err, customers.ErrMissingCustomerID)
	})

	t.Run("When adding a customer with subscriptions and retrieving it", func(t *testing.T) {
		t.Parallel()

		userID := uuid.New().String()
		cusID := uuid.New().String()
		priceID := uuid.New().String()
		subID := uuid.New().String()
		currentPeriodEnd := time.Date(1971, time.April, 31, 00, 00, 00, 00, time.Local)

		params := &customers.CreateCustomerParams{
			ID:         userID,
			CustomerID: cusID,
			FirstName:  "Jane",
			LastName:   "Doe",
		}
		_, err := dir.CreateCustomer(ctx, params)
		assert.Nil(err)

		_, err = dir.StoreSubscriptionWithPrice(ctx, &customers.StoreSubscriptionWithPriceParams{
			Subscription: &customers.StoreSubscriptionParams{
				ID:               subID,
				UserID:           userID,
				CustomerID:       cusID,
				CurrentPeriodEnd: currentPeriodEnd,
				Status:           string(customers.SubscriptionStatusActive),
			},
			Price: &customers.StorePriceParams{
				ID:         priceID,
				UserID:     userID,
				CustomerID: cusID,
				Active:     true,
				Plan:       string(customers.SubscriptionPlanPlusYearly),
			},
		})
		assert.Nil(err)

		// Get customers from db after having stored subscrption
		cus, err := dir.GetCustomerByUserID(ctx, userID)
		assert.Nil(err)
		assert.Equal(len(cus.Subscriptions), 1)
		assert.Equal(cus.Subscriptions[0].CustomerId, cusID)
	})

	t.Run("When adding a customer with default payment method and retrieving it", func(t *testing.T) {
		t.Parallel()
		userID := uuid.New().String()
		cusID := uuid.New().String()
		pmID := uuid.New().String()
		lastFour := "1234"
		expMonth := int32(1)
		expYear := int32(2025)

		params := &customers.CreateCustomerParams{
			ID:         userID,
			CustomerID: cusID,
			FirstName:  "Jane",
			LastName:   "Doe",
		}
		_, err := dir.CreateCustomer(ctx, params)
		assert.Nil(err)

		_, err = dir.StoreDefaultPM(ctx, &customers.StoreDefaultPaymentMethodParams{
			ID:       pmID,
			UserID:   userID,
			LastFour: lastFour,
			ExpMonth: expMonth,
			ExpYear:  expYear,
		})
		assert.Nil(err)

		// Get customers from db after having stored subscrption
		cus, err := dir.GetCustomerByUserID(ctx, userID)
		assert.Nil(err)
		assert.NotNil(cus.DefaultPaymentMethod)
		assert.Equal(cus.DefaultPaymentMethod.Id, pmID)
		assert.Equal(cus.DefaultPaymentMethod.ExpMonth, expMonth)
		assert.Equal(cus.DefaultPaymentMethod.ExpYear, expYear)
		assert.Equal(cus.DefaultPaymentMethod.LastFour, lastFour)
	})
}

func TestGetSubscription(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	dir := getCustomersDirectory(t)

	t.Cleanup(func() {
		err := dir.Close()
		assert.Nil(err)
	})

	t.Run("When inserting and retrieving a subscription with price", func(t *testing.T) {
		userID := "myID"
		cusID := "cusID"
		priceID := "priceID"
		subID := "priceID"
		currentPeriodEnd := time.Date(1971, time.April, 31, 00, 00, 00, 00, time.Local)
		_, err := dir.StoreSubscriptionWithPrice(ctx, &customers.StoreSubscriptionWithPriceParams{
			Subscription: &customers.StoreSubscriptionParams{
				ID:               subID,
				UserID:           userID,
				CustomerID:       cusID,
				CurrentPeriodEnd: currentPeriodEnd,
				Status:           string(customers.SubscriptionStatusActive),
			},
			Price: &customers.StorePriceParams{
				ID:         priceID,
				UserID:     userID,
				CustomerID: cusID,
				Active:     true,
				Plan:       string(customers.SubscriptionPlanPlusYearly),
			},
		})
		assert.Nil(err)

		subByUserID, err := dir.GetSubscriptionByUserID(ctx, userID)
		assert.Nil(err)

		assert.Equal(subByUserID.CurrentPeriodEnd, currentPeriodEnd.Unix())
		assert.Equal(subByUserID.CustomerId, cusID)
		assert.Equal(subByUserID.Id, subID)
		assert.Equal(subByUserID.Status, v1API.SubscriptionStatus_ACTIVE)
		// Price assertions
		assert.Equal(subByUserID.Price.Id, priceID)
		assert.True(subByUserID.Price.Active)
		assert.Equal(subByUserID.Price.Plan, v1API.Plan_UNPAPER_PLUS_YEARLY)
	})
}

func TestStoreSubscription(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	dir := getCustomersDirectory(t)

	t.Cleanup(func() {
		err := dir.Close()
		assert.Nil(err)
	})

	t.Run("When successfully inserting a subscription", func(t *testing.T) {
		t.Parallel()

		unixPeriodEnd := int64(197164800)
		params := &customers.StoreSubscriptionWithPriceParams{
			Subscription: &customers.StoreSubscriptionParams{
				ID:               "myID",
				CustomerID:       "myCustomerID",
				CurrentPeriodEnd: time.Unix(unixPeriodEnd, 0),
				Status:           string(customers.SubscriptionStatusActive),
			},
			Price: &customers.StorePriceParams{
				CustomerID: "myCustomerID",
				ID:         "priceID",
				UserID:     "myID",
				Active:     true,
				Plan:       string(customers.SubscriptionPlanFree),
			},
		}

		// Subscription assertions
		sub, err := dir.StoreSubscriptionWithPrice(ctx, params)
		assert.Nil(err)

		assert.Equal(sub.CurrentPeriodEnd, unixPeriodEnd)
		assert.Equal(sub.Id, params.Subscription.ID)
		assert.Equal(sub.CustomerId, params.Subscription.CustomerID)
		// Price assertions
		assert.Equal(sub.Price.Id, params.Price.ID)
		assert.Equal(sub.Price.Active, params.Price.Active)
		assert.Equal(sub.Price.Plan, v1API.Plan_UNPAPER_FREE)
	})

	t.Run("When providing only subscription", func(t *testing.T) {
		t.Parallel()

		unixPeriodEnd := int64(197164800)
		params := &customers.StoreSubscriptionWithPriceParams{
			Subscription: &customers.StoreSubscriptionParams{
				ID:               "myID",
				CustomerID:       "myCustomerID",
				CurrentPeriodEnd: time.Unix(unixPeriodEnd, 0),
				Status:           string(customers.SubscriptionStatusActive),
			},
		}

		// Subscription assertions
		_, err := dir.StoreSubscriptionWithPrice(ctx, params)
		assert.NotNil(err)
	})

	t.Run("When providing only price", func(t *testing.T) {
		t.Parallel()

		params := &customers.StoreSubscriptionWithPriceParams{
			Price: &customers.StorePriceParams{
				CustomerID: "myCustomerID",
				ID:         "priceID",
				UserID:     "myID",
				Active:     true,
				Plan:       string(customers.SubscriptionPlanFree),
			},
		}

		// Subscription assertions
		_, err := dir.StoreSubscriptionWithPrice(ctx, params)
		assert.NotNil(err)
	})
}

func TestUpdateSubscriptionStatus(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	dir := getCustomersDirectory(t)

	t.Cleanup(func() {
		err := dir.Close()
		assert.Nil(err)
	})

	t.Run("When successfully creating subscription and updating status", func(t *testing.T) {
		t.Parallel()

		unixPeriodEnd := int64(197164800)
		params := &customers.StoreSubscriptionWithPriceParams{
			Subscription: &customers.StoreSubscriptionParams{
				ID:               "myID",
				CustomerID:       "myCustomerID",
				CurrentPeriodEnd: time.Unix(unixPeriodEnd, 0),
				Status:           string(customers.SubscriptionStatusActive),
			},
			Price: &customers.StorePriceParams{
				CustomerID: "myCustomerID",
				ID:         "priceID",
				UserID:     "myID",
				Active:     true,
				Plan:       string(customers.SubscriptionPlanFree),
			},
		}

		_, err := dir.StoreSubscriptionWithPrice(ctx, params)
		assert.Nil(err)

		// Update subscription status to CANCELED
		upParams := customers.UpdateSubscriptionStatusParams{ID: "myID", Status: string(customers.SubscriptionStatusCanceled)}
		updatedSub, err := dir.UpdateSubscriptionStatus(ctx, upParams)
		assert.Nil(err)

		assert.Equal(updatedSub.Status, v1API.SubscriptionStatus_CANCELED)
	})
}

func getCustomersDirectory(t *testing.T) *customers.Directory {
	ws := v1Testing.GetWrappedServer(t)
	dir := customers.NewDirectory(ws.Server.GetDB())

	return dir
}
