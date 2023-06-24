-- name: DoesMatrixUserExist :one
SELECT exists(select 1 from users where name = $1);

-- name: GetUserSpaces :many
SELECT ms.room_id, spaces.space_alias as alias, rs.name, rs.topic, rs.avatar, rs.header, rs.is_profile::boolean as is_profile, spaces.is_default,
CASE WHEN rooms.creator = $1 THEN true ELSE false END as is_owner
FROM membership_state ms 
JOIN spaces ON spaces.room_id = ms.room_id
JOIN rooms ON rooms.room_id = spaces.room_id
LEFT JOIN room_state rs ON rs.room_id = ms.room_id
WHERE ms.user_id = $1
AND ms.membership = 'join'
AND (rs.is_profile is false OR (rs.is_profile and rooms.creator = $1))
ORDER BY rs.is_profile DESC, LOWER(spaces.space_alias) ASC;


-- name: GetJoinedRooms :many
SELECT ms.room_id 
FROM membership_state ms 
WHERE ms.user_id = $1
AND ms.membership = 'join';

-- name: IsUserSpaceMember :one
SELECT exists(
SELECT 1
FROM membership_state ms 
WHERE ms.user_id = $1 
AND ms.room_id = $2 
AND ms.membership = 'join');


-- name: IsAdmin :one
SELECT CASE WHEN admin = 1 THEN TRUE ELSE FALSE END as admin
FROM users
WHERE name = $1;


-- name: CreateUser :one
INSERT INTO users (
  name, password_hash, creation_ts, shadow_banned, approved
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING approved;

-- name: HasUpvoted :one
WITH event AS (
	SELECT events.room_id 
	FROM events WHERE event_id = $1
) SELECT event.room_id, exists(
    SELECT 1, er.event_id
	FROM event_relations er
	JOIN events evs ON evs.event_id = er.relates_to_id
	WHERE er.relation_type = 'm.annotation' 
    AND er.aggregation_key = 'upvote'
    AND evs.sender = $2
    AND er.relates_to_id = $1
) as upvoted,
COALESCE(ve.event_id, '') as event_id
FROM event
LEFT JOIN (
SELECT evs.event_id, evs.room_id
	FROM events evs
	JOIN event_relations er ON evs.event_id = er.event_id
	WHERE er.relation_type = 'm.annotation' 
    AND er.aggregation_key = 'upvote'
    AND evs.sender = $2
    AND er.relates_to_id = $1
) ve ON ve.room_id = event.room_id;

-- name: HasDownvoted :one
WITH event AS (
	SELECT events.room_id 
	FROM events WHERE event_id = $1
) SELECT event.room_id, exists(
    SELECT 1, er.event_id
	FROM event_relations er
	JOIN events evs ON evs.event_id = er.relates_to_id
	WHERE er.relation_type = 'm.annotation' 
    AND er.aggregation_key = 'downvote'
    AND evs.sender = $2
    AND er.relates_to_id = $1
) as downvoted,
COALESCE(ve.event_id, '') as event_id
FROM event
LEFT JOIN (
SELECT evs.event_id, evs.room_id
	FROM events evs
	JOIN event_relations er ON evs.event_id = er.event_id
	WHERE er.relation_type = 'm.annotation' 
    AND er.aggregation_key = 'downvote'
    AND evs.sender = $2
    AND er.relates_to_id = $1
) ve ON ve.room_id = event.room_id;

-- name: UpdatePassword :exec
UPDATE users SET password_hash = $2 WHERE name = $1;
