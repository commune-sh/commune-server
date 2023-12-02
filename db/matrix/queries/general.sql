-- name: GetTablesRowCount :many
SELECT 'spaces' AS table, COUNT(*) AS rows FROM spaces
UNION ALL
SELECT 'users' AS table, COUNT(*) AS users FROM users;

