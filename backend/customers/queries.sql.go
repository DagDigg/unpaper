// Code generated by sqlc. DO NOT EDIT.
// source: queries.sql

package customers

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

const clearCustomerPaymentMethods = `-- name: ClearCustomerPaymentMethods :exec
DELETE FROM stripe_default_payment_methods
WHERE customer_id = $1
`

func (q *Queries) ClearCustomerPaymentMethods(ctx context.Context, customerID string) error {
	_, err := q.db.ExecContext(ctx, clearCustomerPaymentMethods, customerID)
	return err
}

const connectedAccountExists = `-- name: ConnectedAccountExists :one
SELECT EXISTS(SELECT 1 FROM connected_accounts WHERE account_id = $1)
`

func (q *Queries) ConnectedAccountExists(ctx context.Context, accountID string) (bool, error) {
	row := q.db.QueryRowContext(ctx, connectedAccountExists, accountID)
	var exists bool
	err := row.Scan(&exists)
	return exists, err
}

const connectedCustomerExists = `-- name: ConnectedCustomerExists :one
SELECT EXISTS(SELECT 1 FROM connected_customers WHERE customer_id = $1 AND account_id = $2)
`

type ConnectedCustomerExistsParams struct {
	CustomerID string
	AccountID  string
}

func (q *Queries) ConnectedCustomerExists(ctx context.Context, arg ConnectedCustomerExistsParams) (bool, error) {
	row := q.db.QueryRowContext(ctx, connectedCustomerExists, arg.CustomerID, arg.AccountID)
	var exists bool
	err := row.Scan(&exists)
	return exists, err
}

const createCustomer = `-- name: CreateCustomer :one
INSERT INTO customers (id, customer_id, first_name, last_name, account_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING trial_used, id, customer_id, first_name, last_name, account_id
`

type CreateCustomerParams struct {
	ID         string
	CustomerID string
	FirstName  string
	LastName   string
	AccountID  sql.NullString
}

func (q *Queries) CreateCustomer(ctx context.Context, arg CreateCustomerParams) (Customer, error) {
	row := q.db.QueryRowContext(ctx, createCustomer,
		arg.ID,
		arg.CustomerID,
		arg.FirstName,
		arg.LastName,
		arg.AccountID,
	)
	var i Customer
	err := row.Scan(
		&i.TrialUsed,
		&i.ID,
		&i.CustomerID,
		&i.FirstName,
		&i.LastName,
		&i.AccountID,
	)
	return i, err
}

const deleteCustomer = `-- name: DeleteCustomer :exec
DELETE FROM customers
WHERE customer_id = $1
`

func (q *Queries) DeleteCustomer(ctx context.Context, customerID string) error {
	_, err := q.db.ExecContext(ctx, deleteCustomer, customerID)
	return err
}

const getAccountIDFromUserID = `-- name: GetAccountIDFromUserID :one
SELECT account_id
FROM customers
WHERE id = $1
`

func (q *Queries) GetAccountIDFromUserID(ctx context.Context, id string) (sql.NullString, error) {
	row := q.db.QueryRowContext(ctx, getAccountIDFromUserID, id)
	var account_id sql.NullString
	err := row.Scan(&account_id)
	return account_id, err
}

const getAllConnectCustomers = `-- name: GetAllConnectCustomers :many
SELECT account_id, connected_customer_id from connected_customers
WHERE customer_id = $1
`

type GetAllConnectCustomersRow struct {
	AccountID           string
	ConnectedCustomerID string
}

func (q *Queries) GetAllConnectCustomers(ctx context.Context, customerID string) ([]GetAllConnectCustomersRow, error) {
	rows, err := q.db.QueryContext(ctx, getAllConnectCustomers, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetAllConnectCustomersRow
	for rows.Next() {
		var i GetAllConnectCustomersRow
		if err := rows.Scan(&i.AccountID, &i.ConnectedCustomerID); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getConnectedAccount = `-- name: GetConnectedAccount :one
SELECT can_receive_payments, user_id, customer_id, account_id from connected_accounts
WHERE account_id = $1
`

func (q *Queries) GetConnectedAccount(ctx context.Context, accountID string) (ConnectedAccount, error) {
	row := q.db.QueryRowContext(ctx, getConnectedAccount, accountID)
	var i ConnectedAccount
	err := row.Scan(
		&i.CanReceivePayments,
		&i.UserID,
		&i.CustomerID,
		&i.AccountID,
	)
	return i, err
}

const getConnectedAccountByUserID = `-- name: GetConnectedAccountByUserID :one
SELECT can_receive_payments, user_id, customer_id, account_id from connected_accounts
WHERE user_id = $1
`

func (q *Queries) GetConnectedAccountByUserID(ctx context.Context, userID string) (ConnectedAccount, error) {
	row := q.db.QueryRowContext(ctx, getConnectedAccountByUserID, userID)
	var i ConnectedAccount
	err := row.Scan(
		&i.CanReceivePayments,
		&i.UserID,
		&i.CustomerID,
		&i.AccountID,
	)
	return i, err
}

const getConnectedCustomerByCustomerID = `-- name: GetConnectedCustomerByCustomerID :one
SELECT user_id, customer_id, connected_customer_id, account_id from connected_customers
WHERE customer_id = $1
AND account_id = $2
`

type GetConnectedCustomerByCustomerIDParams struct {
	CustomerID string
	AccountID  string
}

func (q *Queries) GetConnectedCustomerByCustomerID(ctx context.Context, arg GetConnectedCustomerByCustomerIDParams) (ConnectedCustomer, error) {
	row := q.db.QueryRowContext(ctx, getConnectedCustomerByCustomerID, arg.CustomerID, arg.AccountID)
	var i ConnectedCustomer
	err := row.Scan(
		&i.UserID,
		&i.CustomerID,
		&i.ConnectedCustomerID,
		&i.AccountID,
	)
	return i, err
}

const getConnectedCustomerByUserID = `-- name: GetConnectedCustomerByUserID :one
SELECT user_id, customer_id, connected_customer_id, account_id from connected_customers
WHERE user_id = $1
AND account_id = $2
`

type GetConnectedCustomerByUserIDParams struct {
	UserID    string
	AccountID string
}

func (q *Queries) GetConnectedCustomerByUserID(ctx context.Context, arg GetConnectedCustomerByUserIDParams) (ConnectedCustomer, error) {
	row := q.db.QueryRowContext(ctx, getConnectedCustomerByUserID, arg.UserID, arg.AccountID)
	var i ConnectedCustomer
	err := row.Scan(
		&i.UserID,
		&i.CustomerID,
		&i.ConnectedCustomerID,
		&i.AccountID,
	)
	return i, err
}

const getCustomerByUserID = `-- name: GetCustomerByUserID :one
SELECT trial_used, id, customer_id, first_name, last_name, account_id from customers
WHERE id = $1
`

func (q *Queries) GetCustomerByUserID(ctx context.Context, id string) (Customer, error) {
	row := q.db.QueryRowContext(ctx, getCustomerByUserID, id)
	var i Customer
	err := row.Scan(
		&i.TrialUsed,
		&i.ID,
		&i.CustomerID,
		&i.FirstName,
		&i.LastName,
		&i.AccountID,
	)
	return i, err
}

const getCustomerUserID = `-- name: GetCustomerUserID :one
SELECT id from customers
WHERE customer_id = $1
`

func (q *Queries) GetCustomerUserID(ctx context.Context, customerID string) (string, error) {
	row := q.db.QueryRowContext(ctx, getCustomerUserID, customerID)
	var id string
	err := row.Scan(&id)
	return id, err
}

const getDefaultPMByUserID = `-- name: GetDefaultPMByUserID :one
SELECT exp_month, exp_year, is_default, id, last_four, user_id, customer_id FROM stripe_default_payment_methods
WHERE (user_id = $1 AND is_default = true)
`

func (q *Queries) GetDefaultPMByUserID(ctx context.Context, userID string) (StripeDefaultPaymentMethod, error) {
	row := q.db.QueryRowContext(ctx, getDefaultPMByUserID, userID)
	var i StripeDefaultPaymentMethod
	err := row.Scan(
		&i.ExpMonth,
		&i.ExpYear,
		&i.IsDefault,
		&i.ID,
		&i.LastFour,
		&i.UserID,
		&i.CustomerID,
	)
	return i, err
}

const getPriceByCustomerID = `-- name: GetPriceByCustomerID :one
SELECT customer_id, id, user_id, plan, active from stripe_prices
WHERE customer_id = $1
`

func (q *Queries) GetPriceByCustomerID(ctx context.Context, customerID string) (StripePrice, error) {
	row := q.db.QueryRowContext(ctx, getPriceByCustomerID, customerID)
	var i StripePrice
	err := row.Scan(
		&i.CustomerID,
		&i.ID,
		&i.UserID,
		&i.Plan,
		&i.Active,
	)
	return i, err
}

const getPriceByUserID = `-- name: GetPriceByUserID :one
SELECT customer_id, id, user_id, plan, active from stripe_prices
WHERE user_id = $1
`

func (q *Queries) GetPriceByUserID(ctx context.Context, userID string) (StripePrice, error) {
	row := q.db.QueryRowContext(ctx, getPriceByUserID, userID)
	var i StripePrice
	err := row.Scan(
		&i.CustomerID,
		&i.ID,
		&i.UserID,
		&i.Plan,
		&i.Active,
	)
	return i, err
}

const getRoomSubscriptionForUserID = `-- name: GetRoomSubscriptionForUserID :one
SELECT latest_invoice, current_period_end, customer_id, connected_customer_id, account_id, id, status, room_id, room_subscription_type, user_id FROM room_subscriptions
WHERE room_id = $1 AND user_id = $2
`

type GetRoomSubscriptionForUserIDParams struct {
	RoomID string
	UserID string
}

func (q *Queries) GetRoomSubscriptionForUserID(ctx context.Context, arg GetRoomSubscriptionForUserIDParams) (RoomSubscription, error) {
	row := q.db.QueryRowContext(ctx, getRoomSubscriptionForUserID, arg.RoomID, arg.UserID)
	var i RoomSubscription
	err := row.Scan(
		&i.LatestInvoice,
		&i.CurrentPeriodEnd,
		&i.CustomerID,
		&i.ConnectedCustomerID,
		&i.AccountID,
		&i.ID,
		&i.Status,
		&i.RoomID,
		&i.RoomSubscriptionType,
		&i.UserID,
	)
	return i, err
}

const getRoomSubscriptionsForUserID = `-- name: GetRoomSubscriptionsForUserID :many
SELECT latest_invoice, current_period_end, customer_id, connected_customer_id, account_id, id, status, room_id, room_subscription_type, user_id FROM room_subscriptions
WHERE user_id = $1
`

func (q *Queries) GetRoomSubscriptionsForUserID(ctx context.Context, userID string) ([]RoomSubscription, error) {
	rows, err := q.db.QueryContext(ctx, getRoomSubscriptionsForUserID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []RoomSubscription
	for rows.Next() {
		var i RoomSubscription
		if err := rows.Scan(
			&i.LatestInvoice,
			&i.CurrentPeriodEnd,
			&i.CustomerID,
			&i.ConnectedCustomerID,
			&i.AccountID,
			&i.ID,
			&i.Status,
			&i.RoomID,
			&i.RoomSubscriptionType,
			&i.UserID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getSubscription = `-- name: GetSubscription :one
SELECT current_period_end, latest_invoice, id, user_id, customer_id, status from stripe_subscriptions
WHERE id = $1
`

func (q *Queries) GetSubscription(ctx context.Context, id string) (StripeSubscription, error) {
	row := q.db.QueryRowContext(ctx, getSubscription, id)
	var i StripeSubscription
	err := row.Scan(
		&i.CurrentPeriodEnd,
		&i.LatestInvoice,
		&i.ID,
		&i.UserID,
		&i.CustomerID,
		&i.Status,
	)
	return i, err
}

const getSubscriptionByCustomerID = `-- name: GetSubscriptionByCustomerID :one
SELECT current_period_end, latest_invoice, id, user_id, customer_id, status from stripe_subscriptions
WHERE customer_id = $1
`

func (q *Queries) GetSubscriptionByCustomerID(ctx context.Context, customerID string) (StripeSubscription, error) {
	row := q.db.QueryRowContext(ctx, getSubscriptionByCustomerID, customerID)
	var i StripeSubscription
	err := row.Scan(
		&i.CurrentPeriodEnd,
		&i.LatestInvoice,
		&i.ID,
		&i.UserID,
		&i.CustomerID,
		&i.Status,
	)
	return i, err
}

const getSubscriptionByUserID = `-- name: GetSubscriptionByUserID :one
SELECT current_period_end, latest_invoice, id, user_id, customer_id, status from stripe_subscriptions
WHERE user_id = $1
`

func (q *Queries) GetSubscriptionByUserID(ctx context.Context, userID string) (StripeSubscription, error) {
	row := q.db.QueryRowContext(ctx, getSubscriptionByUserID, userID)
	var i StripeSubscription
	err := row.Scan(
		&i.CurrentPeriodEnd,
		&i.LatestInvoice,
		&i.ID,
		&i.UserID,
		&i.CustomerID,
		&i.Status,
	)
	return i, err
}

const getUserIDsSubscribedToRoom = `-- name: GetUserIDsSubscribedToRoom :many
SELECT user_id FROM room_subscriptions
WHERE room_id = $1
`

func (q *Queries) GetUserIDsSubscribedToRoom(ctx context.Context, roomID string) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getUserIDsSubscribedToRoom, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var user_id string
		if err := rows.Scan(&user_id); err != nil {
			return nil, err
		}
		items = append(items, user_id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const hasCustomerUsedTrial = `-- name: HasCustomerUsedTrial :one
SELECT trial_used from customers
WHERE customer_id = $1
`

func (q *Queries) HasCustomerUsedTrial(ctx context.Context, customerID string) (sql.NullBool, error) {
	row := q.db.QueryRowContext(ctx, hasCustomerUsedTrial, customerID)
	var trial_used sql.NullBool
	err := row.Scan(&trial_used)
	return trial_used, err
}

const setAccountIDFromUserID = `-- name: SetAccountIDFromUserID :one
UPDATE customers 
SET account_id = $1
WHERE id = $2
RETURNING trial_used, id, customer_id, first_name, last_name, account_id
`

type SetAccountIDFromUserIDParams struct {
	AccountID sql.NullString
	ID        string
}

func (q *Queries) SetAccountIDFromUserID(ctx context.Context, arg SetAccountIDFromUserIDParams) (Customer, error) {
	row := q.db.QueryRowContext(ctx, setAccountIDFromUserID, arg.AccountID, arg.ID)
	var i Customer
	err := row.Scan(
		&i.TrialUsed,
		&i.ID,
		&i.CustomerID,
		&i.FirstName,
		&i.LastName,
		&i.AccountID,
	)
	return i, err
}

const storeConnectedAccount = `-- name: StoreConnectedAccount :one
INSERT INTO connected_accounts (user_id, customer_id, account_id, can_receive_payments)
VALUES ($1, $2, $3, $4)
RETURNING can_receive_payments, user_id, customer_id, account_id
`

type StoreConnectedAccountParams struct {
	UserID             string
	CustomerID         string
	AccountID          string
	CanReceivePayments bool
}

func (q *Queries) StoreConnectedAccount(ctx context.Context, arg StoreConnectedAccountParams) (ConnectedAccount, error) {
	row := q.db.QueryRowContext(ctx, storeConnectedAccount,
		arg.UserID,
		arg.CustomerID,
		arg.AccountID,
		arg.CanReceivePayments,
	)
	var i ConnectedAccount
	err := row.Scan(
		&i.CanReceivePayments,
		&i.UserID,
		&i.CustomerID,
		&i.AccountID,
	)
	return i, err
}

const storeConnectedCustomer = `-- name: StoreConnectedCustomer :one
INSERT INTO connected_customers (user_id, customer_id, connected_customer_id, account_id)
VALUES ($1, $2, $3, $4)
RETURNING user_id, customer_id, connected_customer_id, account_id
`

type StoreConnectedCustomerParams struct {
	UserID              string
	CustomerID          string
	ConnectedCustomerID string
	AccountID           string
}

func (q *Queries) StoreConnectedCustomer(ctx context.Context, arg StoreConnectedCustomerParams) (ConnectedCustomer, error) {
	row := q.db.QueryRowContext(ctx, storeConnectedCustomer,
		arg.UserID,
		arg.CustomerID,
		arg.ConnectedCustomerID,
		arg.AccountID,
	)
	var i ConnectedCustomer
	err := row.Scan(
		&i.UserID,
		&i.CustomerID,
		&i.ConnectedCustomerID,
		&i.AccountID,
	)
	return i, err
}

const storeDefaultPaymentMethod = `-- name: StoreDefaultPaymentMethod :one
INSERT INTO stripe_default_payment_methods (id, user_id, customer_id, last_four, exp_month, exp_year)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (customer_id)
DO UPDATE SET
last_four = EXCLUDED.last_four,
exp_month = EXCLUDED.exp_month,
exp_year = EXCLUDED.exp_year,
id = EXCLUDED.id
RETURNING exp_month, exp_year, is_default, id, last_four, user_id, customer_id
`

type StoreDefaultPaymentMethodParams struct {
	ID         string
	UserID     string
	CustomerID string
	LastFour   string
	ExpMonth   int32
	ExpYear    int32
}

func (q *Queries) StoreDefaultPaymentMethod(ctx context.Context, arg StoreDefaultPaymentMethodParams) (StripeDefaultPaymentMethod, error) {
	row := q.db.QueryRowContext(ctx, storeDefaultPaymentMethod,
		arg.ID,
		arg.UserID,
		arg.CustomerID,
		arg.LastFour,
		arg.ExpMonth,
		arg.ExpYear,
	)
	var i StripeDefaultPaymentMethod
	err := row.Scan(
		&i.ExpMonth,
		&i.ExpYear,
		&i.IsDefault,
		&i.ID,
		&i.LastFour,
		&i.UserID,
		&i.CustomerID,
	)
	return i, err
}

const storePrice = `-- name: StorePrice :one
INSERT INTO stripe_prices (customer_id, id, user_id, active, plan)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (customer_id)
DO UPDATE SET
active = EXCLUDED.active,
plan = EXCLUDED.plan,
id = EXCLUDED.id
RETURNING customer_id, id, user_id, plan, active
`

type StorePriceParams struct {
	CustomerID string
	ID         string
	UserID     string
	Active     bool
	Plan       string
}

func (q *Queries) StorePrice(ctx context.Context, arg StorePriceParams) (StripePrice, error) {
	row := q.db.QueryRowContext(ctx, storePrice,
		arg.CustomerID,
		arg.ID,
		arg.UserID,
		arg.Active,
		arg.Plan,
	)
	var i StripePrice
	err := row.Scan(
		&i.CustomerID,
		&i.ID,
		&i.UserID,
		&i.Plan,
		&i.Active,
	)
	return i, err
}

const storeRoomSubscription = `-- name: StoreRoomSubscription :one
INSERT INTO room_subscriptions (id, user_id, customer_id, connected_customer_id, account_id, current_period_end, status, room_id, room_subscription_type, latest_invoice)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (user_id, room_id)
DO UPDATE SET
current_period_end = EXCLUDED.current_period_end,
status = EXCLUDED.status,
room_subscription_type = EXCLUDED.room_subscription_type,
latest_invoice = EXCLUDED.latest_invoice
RETURNING latest_invoice, current_period_end, customer_id, connected_customer_id, account_id, id, status, room_id, room_subscription_type, user_id
`

type StoreRoomSubscriptionParams struct {
	ID                   string
	UserID               string
	CustomerID           string
	ConnectedCustomerID  string
	AccountID            string
	CurrentPeriodEnd     sql.NullTime
	Status               string
	RoomID               string
	RoomSubscriptionType string
	LatestInvoice        json.RawMessage
}

func (q *Queries) StoreRoomSubscription(ctx context.Context, arg StoreRoomSubscriptionParams) (RoomSubscription, error) {
	row := q.db.QueryRowContext(ctx, storeRoomSubscription,
		arg.ID,
		arg.UserID,
		arg.CustomerID,
		arg.ConnectedCustomerID,
		arg.AccountID,
		arg.CurrentPeriodEnd,
		arg.Status,
		arg.RoomID,
		arg.RoomSubscriptionType,
		arg.LatestInvoice,
	)
	var i RoomSubscription
	err := row.Scan(
		&i.LatestInvoice,
		&i.CurrentPeriodEnd,
		&i.CustomerID,
		&i.ConnectedCustomerID,
		&i.AccountID,
		&i.ID,
		&i.Status,
		&i.RoomID,
		&i.RoomSubscriptionType,
		&i.UserID,
	)
	return i, err
}

const storeSubscription = `-- name: StoreSubscription :one
INSERT INTO stripe_subscriptions (id, user_id, customer_id, current_period_end, status, latest_invoice)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (customer_id)
DO UPDATE SET
id = EXCLUDED.id,
customer_id = EXCLUDED.customer_id,
current_period_end = EXCLUDED.current_period_end,
status = EXCLUDED.status
RETURNING current_period_end, latest_invoice, id, user_id, customer_id, status
`

type StoreSubscriptionParams struct {
	ID               string
	UserID           string
	CustomerID       string
	CurrentPeriodEnd time.Time
	Status           string
	LatestInvoice    json.RawMessage
}

func (q *Queries) StoreSubscription(ctx context.Context, arg StoreSubscriptionParams) (StripeSubscription, error) {
	row := q.db.QueryRowContext(ctx, storeSubscription,
		arg.ID,
		arg.UserID,
		arg.CustomerID,
		arg.CurrentPeriodEnd,
		arg.Status,
		arg.LatestInvoice,
	)
	var i StripeSubscription
	err := row.Scan(
		&i.CurrentPeriodEnd,
		&i.LatestInvoice,
		&i.ID,
		&i.UserID,
		&i.CustomerID,
		&i.Status,
	)
	return i, err
}

const updateConnectedAccountCanReceivePayments = `-- name: UpdateConnectedAccountCanReceivePayments :one
UPDATE connected_accounts
SET can_receive_payments = $1
WHERE account_id = $2
RETURNING can_receive_payments, user_id, customer_id, account_id
`

type UpdateConnectedAccountCanReceivePaymentsParams struct {
	CanReceivePayments bool
	AccountID          string
}

func (q *Queries) UpdateConnectedAccountCanReceivePayments(ctx context.Context, arg UpdateConnectedAccountCanReceivePaymentsParams) (ConnectedAccount, error) {
	row := q.db.QueryRowContext(ctx, updateConnectedAccountCanReceivePayments, arg.CanReceivePayments, arg.AccountID)
	var i ConnectedAccount
	err := row.Scan(
		&i.CanReceivePayments,
		&i.UserID,
		&i.CustomerID,
		&i.AccountID,
	)
	return i, err
}

const updateSubscriptionStatus = `-- name: UpdateSubscriptionStatus :one
UPDATE stripe_subscriptions
SET
status = $1
WHERE id = $2
RETURNING current_period_end, latest_invoice, id, user_id, customer_id, status
`

type UpdateSubscriptionStatusParams struct {
	Status string
	ID     string
}

func (q *Queries) UpdateSubscriptionStatus(ctx context.Context, arg UpdateSubscriptionStatusParams) (StripeSubscription, error) {
	row := q.db.QueryRowContext(ctx, updateSubscriptionStatus, arg.Status, arg.ID)
	var i StripeSubscription
	err := row.Scan(
		&i.CurrentPeriodEnd,
		&i.LatestInvoice,
		&i.ID,
		&i.UserID,
		&i.CustomerID,
		&i.Status,
	)
	return i, err
}

const useTrial = `-- name: UseTrial :exec
UPDATE customers 
SET
trial_used = true
WHERE customer_id = $1
`

func (q *Queries) UseTrial(ctx context.Context, customerID string) error {
	_, err := q.db.ExecContext(ctx, useTrial, customerID)
	return err
}
