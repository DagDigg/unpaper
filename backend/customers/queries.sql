-- name: CreateCustomer :one
INSERT INTO customers (id, customer_id, first_name, last_name, account_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteCustomer :exec
DELETE FROM customers
WHERE customer_id = $1;

-- name: GetCustomerUserID :one
SELECT id from customers
WHERE customer_id = $1;

-- name: GetCustomerByUserID :one
SELECT * from customers
WHERE id = $1;

-- name: StoreConnectedCustomer :one
INSERT INTO connected_customers (user_id, customer_id, connected_customer_id, account_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetConnectedCustomerByUserID :one
SELECT * from connected_customers
WHERE user_id = $1
AND account_id = $2;

-- name: GetConnectedCustomerByCustomerID :one
SELECT * from connected_customers
WHERE customer_id = $1
AND account_id = $2;

-- name: ConnectedCustomerExists :one
SELECT EXISTS(SELECT 1 FROM connected_customers WHERE customer_id = $1 AND account_id = $2);

-- name: GetAllConnectCustomers :many
SELECT account_id, connected_customer_id from connected_customers
WHERE customer_id = $1;

-- name: StoreSubscription :one
INSERT INTO stripe_subscriptions (id, user_id, customer_id, current_period_end, status, latest_invoice)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (customer_id)
DO UPDATE SET
id = EXCLUDED.id,
customer_id = EXCLUDED.customer_id,
current_period_end = EXCLUDED.current_period_end,
status = EXCLUDED.status
RETURNING *;

-- name: GetSubscription :one
SELECT * from stripe_subscriptions
WHERE id = $1;

-- name: GetSubscriptionByUserID :one
SELECT * from stripe_subscriptions
WHERE user_id = $1;

-- name: GetSubscriptionByCustomerID :one
SELECT * from stripe_subscriptions
WHERE customer_id = $1;

-- name: UpdateSubscriptionStatus :one
UPDATE stripe_subscriptions
SET
status = $1
WHERE id = $2
RETURNING *;

-- name: StorePrice :one
INSERT INTO stripe_prices (customer_id, id, user_id, active, plan)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (customer_id)
DO UPDATE SET
active = EXCLUDED.active,
plan = EXCLUDED.plan,
id = EXCLUDED.id
RETURNING *;

-- name: GetPriceByUserID :one
SELECT * from stripe_prices
WHERE user_id = $1;

-- name: GetPriceByCustomerID :one
SELECT * from stripe_prices
WHERE customer_id = $1;

-- name: StoreDefaultPaymentMethod :one
INSERT INTO stripe_default_payment_methods (id, user_id, customer_id, last_four, exp_month, exp_year)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (customer_id)
DO UPDATE SET
last_four = EXCLUDED.last_four,
exp_month = EXCLUDED.exp_month,
exp_year = EXCLUDED.exp_year,
id = EXCLUDED.id
RETURNING *;

-- name: GetDefaultPMByUserID :one
SELECT * FROM stripe_default_payment_methods
WHERE (user_id = $1 AND is_default = true);

-- name: ClearCustomerPaymentMethods :exec
DELETE FROM stripe_default_payment_methods
WHERE customer_id = $1;

-- name: HasCustomerUsedTrial :one
SELECT trial_used from customers
WHERE customer_id = $1;

-- name: UseTrial :exec
UPDATE customers 
SET
trial_used = true
WHERE customer_id = $1;

-- name: GetAccountIDFromUserID :one
SELECT account_id
FROM customers
WHERE id = $1;

-- name: SetAccountIDFromUserID :one
UPDATE customers 
SET account_id = $1
WHERE id = $2
RETURNING *;

-- name: StoreRoomSubscription :one
INSERT INTO room_subscriptions (id, user_id, customer_id, connected_customer_id, account_id, current_period_end, status, room_id, room_subscription_type, latest_invoice)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (user_id, room_id)
DO UPDATE SET
current_period_end = EXCLUDED.current_period_end,
status = EXCLUDED.status,
room_subscription_type = EXCLUDED.room_subscription_type,
latest_invoice = EXCLUDED.latest_invoice
RETURNING *;

-- name: GetRoomSubscriptionForUserID :one
SELECT * FROM room_subscriptions
WHERE room_id = $1 AND user_id = $2;

-- name: GetRoomSubscriptionsForUserID :many
SELECT * FROM room_subscriptions
WHERE user_id = $1;

-- name: GetUserIDsSubscribedToRoom :many
SELECT user_id FROM room_subscriptions
WHERE room_id = $1;

-- name: GetConnectedAccount :one
SELECT * from connected_accounts
WHERE account_id = $1;

-- name: GetConnectedAccountByUserID :one
SELECT * from connected_accounts
WHERE user_id = $1;

-- name: StoreConnectedAccount :one
INSERT INTO connected_accounts (user_id, customer_id, account_id, can_receive_payments)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ConnectedAccountExists :one
SELECT EXISTS(SELECT 1 FROM connected_accounts WHERE account_id = $1);

-- name: UpdateConnectedAccountCanReceivePayments :one
UPDATE connected_accounts
SET can_receive_payments = $1
WHERE account_id = $2
RETURNING *;