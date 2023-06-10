-- name: GetReactionEventID :one
SELECT er.event_id
FROM event_relations er 
JOIN events ON events.event_id = er.event_id
WHERE events.type = 'm.reaction'
AND events.room_id = $1
AND er.relates_to_id = $2
AND events.sender = $3
AND er.aggregation_key = $4;
