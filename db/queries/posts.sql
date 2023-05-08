-- name: GetUserPosts :many
SELECT posts.slug, posts.content FROM posts
LEFT JOIN users ON posts.user_id = users.id
WHERE users.username = $1 LIMIT 25;
