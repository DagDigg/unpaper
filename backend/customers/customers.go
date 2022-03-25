package customers

import (
	"context"
	"database/sql"
	"fmt"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/Masterminds/squirrel"

	// pgx is a postgres driver
	_ "github.com/jackc/pgx/v4/stdlib"
)

// Directory is the directory which operates on db table 'users'
type Directory struct {
	// querier is an interface containing all of the
	// directory methods. Must be created with customers.NewDirectory(db)
	querier Querier
	db      *sql.DB
	sb      squirrel.StatementBuilderType
}

// NewDirectory creates a new users directory
func NewDirectory(db *sql.DB) *Directory {
	return &Directory{db: db, querier: New(db)}
}

// Close closes Directory database connection
func (d Directory) Close() error {
	return d.db.Close()
}

var (
	// ErrMissingUserID is an error representing the absence of
	// the userID
	ErrMissingUserID = fmt.Errorf("missing userID")
	// ErrMissingPriceID is an error representing the absence of
	// the priceID
	ErrMissingPriceID = fmt.Errorf("missing priceID")
	// ErrMissingCustomerID is an error representing the absence of
	// the customerID
	ErrMissingCustomerID = fmt.Errorf("missing customerID")
	// ErrNoCustomer is an error returned when a customer
	// couldn't be found on database
	ErrNoCustomer = fmt.Errorf("no customer found")
)

// SubscriptionStatus refers to the status of the subscription
type SubscriptionStatus string

// RoomSubscriptionType refers to the type of room subscription
type RoomSubscriptionType string

// SubscriptionPlan refers to the plan of the subscription
type SubscriptionPlan string

const (
	// SubscriptionStatusActive refers to an active subscription
	SubscriptionStatusActive SubscriptionStatus = "active"
	// SubscriptionStatusCanceled refers to a canceled subscription
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	// SubscriptionStatusIncomplete refers to an incomplete subscription
	SubscriptionStatusIncomplete SubscriptionStatus = "incomplete"
	// SubscriptionStatusIncompleteExpired refers to an incomplete_expired subscription
	SubscriptionStatusIncompleteExpired SubscriptionStatus = "incomplete_expired"
	// SubscriptionStatusPastDue refers to a past_due subscription
	SubscriptionStatusPastDue SubscriptionStatus = "past_due"
	// SubscriptionStatusTrialing refers to a trialing subscription
	SubscriptionStatusTrialing SubscriptionStatus = "trialing"
	// SubscriptionStatusUnpaid refers to an unpaid subscription
	SubscriptionStatusUnpaid SubscriptionStatus = "unpaid"
	// RoomSubscriptionTypeOneTime refers to a one_time purchase
	RoomSubscriptionTypeOneTime RoomSubscriptionType = "one_time"
	// RoomSubscriptionTypeSubscriptionMonthly refers to a monthly recurring purchase
	RoomSubscriptionTypeSubscriptionMonthly RoomSubscriptionType = "subscription_monthly"
	// SubscriptionPlanFree refers to a free plan
	SubscriptionPlanFree SubscriptionPlan = "free"
	// SubscriptionPlanPlusMonthly refers to a plus monthly paid subscription
	SubscriptionPlanPlusMonthly SubscriptionPlan = "plus_monthly"
	// SubscriptionPlanPlusYearly refers to a plus yearly paid subscription
	SubscriptionPlanPlusYearly SubscriptionPlan = "plus_yearly"
)

// CreateCustomer inserts a customer into db. It only stores it.
// The creation of the Stripe customer happens on the service.
// The created customer has no attached subscriptions
func (d Directory) CreateCustomer(ctx context.Context, params *CreateCustomerParams) (*v1API.Customer, error) {
	if params.ID == "" {
		return nil, ErrMissingUserID
	}
	if params.CustomerID == "" {
		return nil, ErrMissingCustomerID
	}
	res, err := d.querier.CreateCustomer(ctx, *params)
	if err != nil {
		return nil, fmt.Errorf("error inserting customer into db")
	}

	return pgCustomerToPB(res)
}

// DeleteCustomer deletes a customer by customerID on db
func (d Directory) DeleteCustomer(ctx context.Context, customerID string) error {
	return d.querier.DeleteCustomer(ctx, customerID)
}

// GetCustomerByUserID retrieve the database customer selected by its userID
// and returns the protobuf *Customer value
func (d Directory) GetCustomerByUserID(ctx context.Context, userID string) (*v1API.Customer, error) {
	pgCustomer, err := d.querier.GetCustomerByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNoCustomer
		}
		return nil, fmt.Errorf("error getting customer: %q", err)
	}
	hasSubscriptions := true
	subs, err := getPBSubscriptionByUserID(ctx, d.querier, userID)
	if err != nil {
		if err != sql.ErrNoRows {
			// It's okay if customer has no subscriptions.
			// The consumer should check for them
			return nil, fmt.Errorf("failed to retrieve customer subscription: %q", err)
		}
		hasSubscriptions = false
	}

	pbCus, err := pgCustomerToPB(pgCustomer)
	if err != nil {
		return nil, fmt.Errorf("error converting PG customer to PB: %q", err)
	}

	if hasSubscriptions {
		// Attach subscriptions
		pbCus.Subscriptions = []*v1API.Subscription{subs}

		price, err := getPBPriceByCustomerID(ctx, d.querier, pbCus.CustomerId)
		if err != nil {
			return nil, fmt.Errorf("failure retrieving price for customer on db: %q", err)
		}
		// Attach priceID.
		// TODO: remove 'active' hardcoded value
		pbCus.Subscriptions[0].Price = price
	}

	// Retrieve default payment method
	pm, err := getPBDefaultPMByUserID(ctx, d.querier, userID)
	hasDefaultPM := true
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("failure retrieving default payment method for customer on db: %q", err)
		}
		hasDefaultPM = false
	}
	if hasDefaultPM {
		pbCus.DefaultPaymentMethod = pm
	}

	return pbCus, nil
}

// GetCustomerUserID returns the userID associated with the customer
func (d Directory) GetCustomerUserID(ctx context.Context, customerID string) (string, error) {
	return d.querier.GetCustomerUserID(ctx, customerID)
}

// StoreConnectedCustomer saves connected customer data to db
func (d Directory) StoreConnectedCustomer(ctx context.Context, params *StoreConnectedCustomerParams) (*v1API.ConnectedCustomer, error) {
	res, err := d.querier.StoreConnectedCustomer(ctx, *params)
	if err != nil {
		return nil, err
	}

	return pgConnectedCustomerToPB(res), nil
}

// GetConnectedCustomerByUserID returns connected customer data by user id
func (d Directory) GetConnectedCustomerByUserID(ctx context.Context, params *GetConnectedCustomerByUserIDParams) (*v1API.ConnectedCustomer, error) {
	res, err := d.querier.GetConnectedCustomerByUserID(ctx, *params)
	if err != nil {
		return nil, err
	}

	return pgConnectedCustomerToPB(res), nil
}

// GetConnectedCustomerByCustomerID returns connected customer data by customer id
func (d Directory) GetConnectedCustomerByCustomerID(ctx context.Context, params *GetConnectedCustomerByCustomerIDParams) (*v1API.ConnectedCustomer, error) {
	res, err := d.querier.GetConnectedCustomerByCustomerID(ctx, *params)
	if err != nil {
		return nil, err
	}

	return pgConnectedCustomerToPB(res), nil
}

// ConnectedCustomerExists returns whether a platform `customer_id` AND `account_id` exists on the `connected_account` table.
// They must be on the same row
func (d Directory) ConnectedCustomerExists(ctx context.Context, params *ConnectedCustomerExistsParams) (bool, error) {
	return d.querier.ConnectedCustomerExists(ctx, *params)
}

// GetAllConnectCustomers returns every `account_id` where the passed platform customer exists
func (d Directory) GetAllConnectCustomers(ctx context.Context, customerID string) ([]GetAllConnectCustomersRow, error) {
	return d.querier.GetAllConnectCustomers(ctx, customerID)
}

// StoreSubscriptionWithPriceParams parameters for storing subscription with price
type StoreSubscriptionWithPriceParams struct {
	Subscription *StoreSubscriptionParams
	Price        *StorePriceParams
	WithTrial    bool
}

// StoreSubscriptionWithPrice inserts into db a subscription for a customer
func (d Directory) StoreSubscriptionWithPrice(ctx context.Context, params *StoreSubscriptionWithPriceParams) (*v1API.Subscription, error) {
	if params.Subscription == nil {
		return nil, fmt.Errorf("subscription param is required")
	}
	if params.Price == nil {
		return nil, fmt.Errorf("price param is required")
	}
	pgSub, err := d.querier.StoreSubscription(ctx, *params.Subscription)
	if err != nil {
		return nil, fmt.Errorf("failed to store subscription on db: %q", err)
	}
	pbSub, err := pgSubscriptionToPB(pgSub)
	if err != nil {
		return nil, fmt.Errorf("failed to convert postgres subscription to protobuf: %q", err)
	}
	pgPrice, err := d.querier.StorePrice(ctx, *params.Price)
	if err != nil {
		return nil, fmt.Errorf("failed to store price on db: %q", err)
	}
	pbPrice, err := pgPriceToPB(pgPrice)
	if err != nil {
		return nil, fmt.Errorf("failed to convert postgres price to protobuf: %q", err)
	}
	if params.WithTrial == true {
		err = d.querier.UseTrial(ctx, params.Subscription.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("failed to update trial_used: %q", err)
		}
	}

	// Attach price
	pbSub.Price = pbPrice
	return pbSub, nil
}

// GetSubscriptionByID retrieves from database a subscription by its subscription ID.
// It currently returns only the subscription without its price
func (d Directory) GetSubscriptionByID(ctx context.Context, id string) (*v1API.Subscription, error) {
	subs, err := d.querier.GetSubscription(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve subscription: %q", err)
	}

	return pgSubscriptionToPB(subs)
}

func (d Directory) GetSubscriptionByUserID(ctx context.Context, userID string) (*v1API.Subscription, error) {
	return getPBSubscriptionByUserID(ctx, d.querier, userID)
}

// GetSubscriptionByCustomerID retrieves from database the customer subscription by its customer ID
func (d Directory) GetSubscriptionByCustomerID(ctx context.Context, id string) (*v1API.Subscription, error) {
	return getPBSubscriptionByCustomerID(ctx, d.querier, id)
}

func getPBSubscriptionByCustomerID(ctx context.Context, q Querier, id string) (*v1API.Subscription, error) {
	subs, err := q.GetSubscriptionByCustomerID(ctx, id)
	if err != nil {
		return nil, err
	}

	return pgSubscriptionToPB(subs)
}

// UpdateSubscriptionStatus updates the subscription status
func (d Directory) UpdateSubscriptionStatus(ctx context.Context, params UpdateSubscriptionStatusParams) (*v1API.Subscription, error) {
	subs, err := d.querier.UpdateSubscriptionStatus(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription status: %q", err)
	}

	return pgSubscriptionToPB(subs)
}

// StoreDefaultPM saves on database the customer payment method
func (d Directory) StoreDefaultPM(ctx context.Context, params *StoreDefaultPaymentMethodParams) (*v1API.PaymentMethod, error) {
	pm, err := d.querier.StoreDefaultPaymentMethod(ctx, *params)
	if err != nil {
		return nil, fmt.Errorf("failed to store payment method: %q", err)
	}

	return pgPaymentMethodToPB(pm)
}

// ClearCustomerPaymentMethods deletes all payment methods saved on database
// attached to the passed customerID. It returns an error if encountered during the transaction
func (d Directory) ClearCustomerPaymentMethods(ctx context.Context, customerID string) error {
	err := d.querier.ClearCustomerPaymentMethods(ctx, customerID)
	if err != nil {
		return err
	}

	return nil
}

// HasCustomerUsedTrial returns whether trial_used column is true or false
func (d Directory) HasCustomerUsedTrial(ctx context.Context, customerID string) (bool, error) {
	res, err := d.querier.HasCustomerUsedTrial(ctx, customerID)
	if err != nil {
		return false, fmt.Errorf("error retrieving trial_used from customers: %v", err)
	}

	return res.Bool, nil
}

// StorePrice saves price on db
func (d Directory) StorePrice(ctx context.Context, params StorePriceParams) (*v1API.Price, error) {
	pgPrice, err := d.querier.StorePrice(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to store price on db: %q", err)
	}

	return pgPriceToPB(pgPrice)
}

// GetPriceByCustomerID returns the Stripe price by its customer ID
func (d Directory) GetPriceByCustomerID(ctx context.Context, id string) (*v1API.Price, error) {
	return getPBPriceByCustomerID(ctx, d.querier, id)
}

// GetConnectedAccount returns the connected account queried by `account_id` PK
func (d Directory) GetConnectedAccount(ctx context.Context, acctID string) (*v1API.ConnectedAccount, error) {
	res, err := d.querier.GetConnectedAccount(ctx, acctID)
	if err != nil {
		return nil, err
	}

	return pgConnectedAccountToPB(res), nil
}

// GetConnectedAccountByUserID returns the connected account queried by `user_id`
func (d Directory) GetConnectedAccountByUserID(ctx context.Context, userID string) (*v1API.ConnectedAccount, error) {
	res, err := d.querier.GetConnectedAccountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return pgConnectedAccountToPB(res), nil
}

// StoreConnectedAccount saves a connected account on db
func (d Directory) StoreConnectedAccount(ctx context.Context, params *StoreConnectedAccountParams) (*v1API.ConnectedAccount, error) {
	res, err := d.querier.StoreConnectedAccount(ctx, *params)
	if err != nil {
		return nil, err
	}

	return pgConnectedAccountToPB(res), nil
}

// ConnectedAccountExists returns whether a row queried by `account_id` exists
func (d Directory) ConnectedAccountExists(ctx context.Context, acctID string) (bool, error) {
	return d.querier.ConnectedAccountExists(ctx, acctID)
}

// UpdateConnectedAccountCanReceivePayments updates the `can_receive_payments` column
func (d Directory) UpdateConnectedAccountCanReceivePayments(ctx context.Context, params *UpdateConnectedAccountCanReceivePaymentsParams) (*v1API.ConnectedAccount, error) {
	res, err := d.querier.UpdateConnectedAccountCanReceivePayments(ctx, *params)
	if err != nil {
		return nil, err
	}

	return pgConnectedAccountToPB(res), nil
}

// GetAccountIDFromUserID retrieves the account_id from customers by user_id
func (d Directory) GetAccountIDFromUserID(ctx context.Context, userID string) (string, error) {
	res, err := d.querier.GetAccountIDFromUserID(ctx, userID)
	if err != nil {
		return "", err
	}
	if !res.Valid {
		return "", nil
	}
	return res.String, nil
}

// SetAccountIDFromUserID updates the customer account_id column by userID
func (d Directory) SetAccountIDFromUserID(ctx context.Context, params *SetAccountIDFromUserIDParams) (*v1API.Customer, error) {
	res, err := d.querier.SetAccountIDFromUserID(ctx, *params)
	if err != nil {
		return nil, err
	}

	return pgCustomerToPB(res)
}

// StoreRoomSubscription stores a room subscription on db
func (d Directory) StoreRoomSubscription(ctx context.Context, params *StoreRoomSubscriptionParams) (*v1API.RoomSubscription, error) {
	res, err := d.querier.StoreRoomSubscription(ctx, *params)
	if err != nil {
		return nil, err
	}

	return pgRoomSubscriptionToPB(res)
}

// GetRoomSubscriptionForUserID retrieves a room subscription associated to the user id
func (d Directory) GetRoomSubscriptionForUserID(ctx context.Context, params *GetRoomSubscriptionForUserIDParams) (*v1API.RoomSubscription, error) {
	res, err := d.querier.GetRoomSubscriptionForUserID(ctx, *params)
	if err != nil {
		return nil, err
	}

	return pgRoomSubscriptionToPB(res)
}

// GetRoomSubscriptionsForUserID retrieves all room subscriptions on db by user id
func (d Directory) GetRoomSubscriptionsForUserID(ctx context.Context, userID string) ([]*v1API.RoomSubscription, error) {
	res, err := d.querier.GetRoomSubscriptionsForUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Iterate and convert postgres subscriptions to protobuf
	subsc := []*v1API.RoomSubscription{}
	for _, s := range res {
		pbSubsc, err := pgRoomSubscriptionToPB(s)
		if err != nil {
			return nil, err
		}
		subsc = append(subsc, pbSubsc)
	}

	return subsc, nil
}

// GetUserIDsSubscribedToRoom returns a list of user ids that have a room subscription in any status
func (d Directory) GetUserIDsSubscribedToRoom(ctx context.Context, roomID string) ([]string, error) {
	res, err := d.querier.GetUserIDsSubscribedToRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func getPBPriceByCustomerID(ctx context.Context, q Querier, customerID string) (*v1API.Price, error) {
	if customerID == "" {
		return nil, ErrMissingCustomerID
	}
	pgPrice, err := q.GetPriceByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving price from db: %q", err)
	}

	pbPlan, err := pgPlanToPB(SubscriptionPlan(pgPrice.Plan))
	if err != nil {
		return nil, err
	}

	return &v1API.Price{
		Id:     pgPrice.ID,
		Active: pgPrice.Active,
		Plan:   pbPlan,
	}, nil
}

func getPBSubscriptionByUserID(ctx context.Context, q Querier, userID string) (*v1API.Subscription, error) {
	pgSubs, err := q.GetSubscriptionByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Propagate error
			return nil, err
		}
		return nil, fmt.Errorf("failed to retrieve subscription: %q", err)
	}
	pgPrice, err := q.GetPriceByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve price: %q", err)
	}

	pbSub, err := pgSubscriptionToPB(pgSubs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert pg subscription to pb: %q", err)
	}
	pbPrice, err := pgPriceToPB(pgPrice)
	if err != nil {
		return nil, fmt.Errorf("failed to convert pg price to pb: %q", err)
	}

	pbSub.Price = pbPrice
	return pbSub, nil
}

func getPBDefaultPMByUserID(ctx context.Context, q Querier, userID string) (*v1API.PaymentMethod, error) {
	pgPM, err := q.GetDefaultPMByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return pgPaymentMethodToPB(pgPM)
}
