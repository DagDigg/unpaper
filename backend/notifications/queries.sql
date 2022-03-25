-- name: GetNotificationByID :one
SELECT n.*, u.username FROM notifications n JOIN users u
ON u.id = n.user_id_who_fired_event
WHERE n.id=$1;

-- name: CreateNotification :one
WITH n AS (
    INSERT INTO notifications 
	(id, user_id_to_notify, user_id_who_fired_event, trigger_id, event_id, date, content)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING *
)
SELECT n.*, u.username
FROM n
JOIN users u ON n.user_id_who_fired_event = u.id;

-- name: GetUnreadNotifications :many
SELECT n.*, u.username FROM notifications n JOIN users u
ON u.id = n.user_id_who_fired_event
WHERE user_id_to_notify=$1 AND read=false ORDER BY date DESC;

-- name: ReadNotification :one
WITH n AS (
	UPDATE notifications ns
	SET read=true 
	WHERE ns.id=$1
	RETURNING *
)
SELECT n.*, u.username
FROM n
JOIN users u ON n.user_id_who_fired_event = u.id;

-- name: NotificationAlreadyExists :one
SELECT EXISTS(
	SELECT id FROM notifications
	WHERE
	user_id_to_notify=$1 AND
	user_id_who_fired_event=$2 AND
	trigger_id=$3 AND
	event_id=$4
);

-- name: GetNotification :one
SELECT n.*, u.username FROM notifications n JOIN users u
ON u.id = n.user_id_who_fired_event
WHERE
user_id_to_notify=$1 AND
user_id_who_fired_event=$2 AND
trigger_id=$3 AND
event_id=$4;


-- name: GetAllNotifications :many
SELECT n.*, u.username FROM notifications n JOIN users u
ON u.id = n.user_id_who_fired_event
WHERE n.user_id_to_notify=$1 AND n.read=true
UNION ALL
SELECT n.*, u.username FROM notifications n JOIN users u
ON u.id = n.user_id_who_fired_event
WHERE n.user_id_to_notify=$1 AND n.read=false
ORDER BY date DESC;