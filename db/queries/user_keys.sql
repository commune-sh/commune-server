-- name: CreateUserKey :exec
INSERT INTO user_keys (
    matrix_user_id, public_key, private_key
) VALUES (
  $1, $2, $3
);
