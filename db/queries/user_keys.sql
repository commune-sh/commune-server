-- name: CreateUserKey :exec
INSERT INTO user_keys (
    matrix_user_id, public_key, private_key
) VALUES (
  $1, $2, $3
);

-- name: GetUserPublicKey :one
SELECT public_key FROM user_keys WHERE matrix_user_id = $1;

-- name: GetUserPrivateKey :one
SELECT private_key  FROM user_keys WHERE matrix_user_id = $1;
