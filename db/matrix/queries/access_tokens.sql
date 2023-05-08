-- name: IsAccessTokenValid :one
SELECT exists(select 1 from access_tokens where user_id = $1 and token = $2);

-- name: CreateAccessToken :one
insert into access_tokens(
    id, user_id, device_id, token
)
SELECT max(id) + 1, $1, $2, $3
FROM access_tokens
RETURNING id;
