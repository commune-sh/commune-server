-- name: DoesMatrixUserExist :one
SELECT exists(select 1 from users where name = $1);

-- name: DoesUsernameExist :one
SELECT exists(select 1 from profiles where user_id = sqlc.narg('username'));

-- name: IsDeactivated :one
SELECT CASE WHEN users.deactivated = 1 THEN TRUE ELSE FALSE END as deactivated
FROM users
JOIN profiles ON profiles.full_user_id = users.name
WHERE profiles.user_id = sqlc.narg('username');

-- name: GetCredentials :one
SELECT users.name as matrix_user_id, 
    profiles.user_id as username, 
    utpid.address as email, 
    CASE WHEN utpid.address IS NULL THEN false ELSE true END as verified,
    users.creation_ts as created_at
FROM users
JOIN profiles ON profiles.full_user_id = users.name
LEFT JOIN user_threepids utpid ON utpid.user_id = users.name
WHERE (profiles.user_id = sqlc.narg('username') OR utpid.address = sqlc.narg('username')) AND users.deactivated = 0;

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


-- name: GetUserCreatedAt :one
SELECT creation_ts FROM users WHERE name = $1;

-- name: VerifyEmail :exec
INSERT INTO user_threepids (user_id, medium, address, validated_at, added_at)
VALUES (sqlc.narg('matrix_user_id'), 'email', sqlc.narg('email'), EXTRACT(epoch FROM CURRENT_TIMESTAMP)::bigint, EXTRACT(epoch FROM CURRENT_TIMESTAMP)::bigint);

-- name: DoesEmailExist :one
SELECT exists(
    select 1 from user_threepids 
    join users ON users.name = user_threepids.user_id
    where address = sqlc.narg('email') 
    AND users.deactivated = 0);

-- name: IsVerifed :one
SELECT EXISTS (
  SELECT 1
  FROM user_threepids
  WHERE user_id = $1
) AS verified;

-- name: GetJoinedRooms :many
SELECT ms.room_id 
FROM membership_state ms 
WHERE ms.user_id = $1
AND ms.membership = 'join';

-- name: GetProfileFollowers :many
SELECT ms.*
FROM membership_state ms 
WHERE ms.user_id != $1
AND ms.room_id = $2
AND (ms.origin_server_ts > sqlc.narg('origin_server_ts') OR sqlc.narg('origin_server_ts') IS NULL)
AND ms.membership = 'join';

-- name: IsUserSpaceMember :one
SELECT exists(
SELECT 1
FROM membership_state ms 
WHERE ms.user_id = $1 
AND ms.room_id = $2 
AND ms.membership = 'join');

-- name: GetDMs :many
SELECT ad.content
FROM account_data ad
WHERE ad.user_id = $1
AND ad.account_data_type = 'm.direct';

-- name: IsAdmin :one
SELECT CASE WHEN admin = 1 THEN TRUE ELSE FALSE END as admin
FROM users
WHERE name = $1;

-- name: UpdateProfilesAvatar :exec
UPDATE profiles SET avatar_url = $2 WHERE full_user_id = $1;

-- name: UpdateUserDirectoryAvatar :exec
UPDATE user_directory SET avatar_url = $2 WHERE user_id = $1;

-- name: UpdateProfilesDisplayName :exec
UPDATE profiles SET displayname = $2 WHERE full_user_id = $1;

-- name: UpdateUserDirectoryDisplayName :exec
UPDATE user_directory SET display_name = $2 WHERE user_id = $1;


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

-- name: DeactivateUser :exec
UPDATE users SET deactivated = 1 WHERE name = sqlc.narg('matrix_user_id');
