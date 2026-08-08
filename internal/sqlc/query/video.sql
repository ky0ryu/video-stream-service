-- name: CreateVideo :one
INSERT INTO videos (
  id, title, description, state, original_filename, stored_filename, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: UpdateVideoState :one
UPDATE videos SET state = $2 WHERE id = $1
RETURNING *;

-- name: GetVideo :one
SELECT * FROM videos
WHERE id = $1;
