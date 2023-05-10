-- name: DoesMatrixUserExist :one
SELECT exists(select 1 from users where name = $1);

-- name: GetUserSpaces :many
SELECT sc.parent_room_alias, rm.room_id, rs.name, rs.topic, rs.avatar, rs.header
FROM room_memberships rm
JOIN space_children sc ON sc.parent_room_id = rm.room_id
LEFT JOIN room_state rs ON rs.room_id = sc.parent_room_id
WHERE rm.user_id = $1
GROUP BY sc.parent_room_alias, rm.room_id, rs.name, rs.topic, rs.avatar, rs.header
HAVING COUNT(CASE WHEN rm.membership = 'join' THEN 1 ELSE NULL END) > 0
   AND COUNT(CASE WHEN rm.membership = 'leave' THEN 1 ELSE NULL END) = 0;

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
