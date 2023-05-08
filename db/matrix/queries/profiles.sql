-- name: GetProfile :one
SELECT displayname, avatar_url from profiles
WHERE user_id = $1 LIMIT 1;

-- name: CreateProfile :one
INSERT INTO profiles (
    user_id, displayname
) VALUES (
  $1, $2
)
RETURNING user_id;
