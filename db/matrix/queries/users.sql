-- name: DoesMatrixUserExist :one
SELECT exists(select 1 from users where name = $1);

-- name: GetUserSpaces :many
SELECT ms.room_id, spaces.space_alias as alias, rs.name, rs.topic, rs.avatar, rs.header
FROM membership_state ms 
JOIN spaces ON spaces.room_id = ms.room_id
LEFT JOIN room_state rs ON rs.room_id = ms.room_id
WHERE ms.user_id = $1
AND ms.membership = 'join'
AND rs.is_profile is false
ORDER BY spaces.space_alias ASC;

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


-- name: CreateUser :one
INSERT INTO users (
  name, password_hash, creation_ts, shadow_banned, approved
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING approved;
