-- name: GetSpaceCount :one
SELECT
    SUM(CASE WHEN space_alias LIKE '@%' THEN 1 ELSE 0 END) AS users,
    SUM(CASE WHEN space_alias NOT LIKE '@%' THEN 1 ELSE 0 END) AS spaces
FROM spaces;

