-- name: ListServicesByApplication :many
SELECT id, application_id, instance_key, version_id, last_successful_version_id, status, created_at, updated_at
FROM service
WHERE application_id = ?
ORDER BY instance_key;

-- name: ServiceByID :one
SELECT id, application_id, instance_key, version_id, last_successful_version_id, status, created_at, updated_at
FROM service
WHERE id = ?;

-- name: ServiceByKey :one
SELECT id, application_id, instance_key, version_id, last_successful_version_id, status, created_at, updated_at
FROM service
WHERE application_id = ? AND instance_key = ?;

-- name: ServiceIDByKey :one
SELECT id
FROM service
WHERE application_id = ? AND instance_key = ?;

-- name: InsertService :exec
INSERT INTO service (
  id, application_id, instance_key, version_id, last_successful_version_id, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateService :exec
UPDATE service
SET version_id = ?, last_successful_version_id = ?, status = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateServiceStatus :exec
UPDATE service
SET status = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateServiceAfterDeploy :exec
UPDATE service
SET status = ?, version_id = ?, last_successful_version_id = ?, updated_at = ?
WHERE id = ?;

-- name: DetachDeploymentServiceRefs :exec
UPDATE deployment
SET service_id = NULL
WHERE service_id = ?;

-- name: DeleteService :exec
DELETE FROM service
WHERE id = ?;

-- name: CountServicesByProject :one
SELECT COUNT(*)
FROM service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN version v ON v.id = s.version_id
LEFT JOIN version lv ON lv.id = s.last_successful_version_id
WHERE a.project_id = ?
  AND (? = '' OR s.application_id = ?)
  AND (? = '' OR s.status = ?)
  AND (? = '' OR a.name LIKE ? OR a.code LIKE ? OR s.instance_key LIKE ? OR v.label LIKE ?);

-- name: ListServicesByProject :many
SELECT s.id, s.application_id, s.instance_key, s.version_id, s.last_successful_version_id, s.status, s.created_at, s.updated_at,
       a.name AS application_name, a.code AS application_code, a.kind AS application_kind,
       v.label AS version_label,
       lv.label AS last_successful_version_label
FROM service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN version v ON v.id = s.version_id
LEFT JOIN version lv ON lv.id = s.last_successful_version_id
WHERE a.project_id = ?
  AND (? = '' OR s.application_id = ?)
  AND (? = '' OR s.status = ?)
  AND (? = '' OR a.name LIKE ? OR a.code LIKE ? OR s.instance_key LIKE ? OR v.label LIKE ?)
ORDER BY s.updated_at DESC, a.name, s.instance_key
LIMIT ? OFFSET ?;

-- name: ServiceListItemByID :one
SELECT s.id, s.application_id, s.instance_key, s.version_id, s.last_successful_version_id, s.status, s.created_at, s.updated_at,
       a.name AS application_name, a.code AS application_code, a.kind AS application_kind,
       v.label AS version_label,
       lv.label AS last_successful_version_label
FROM service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN version v ON v.id = s.version_id
LEFT JOIN version lv ON lv.id = s.last_successful_version_id
WHERE s.id = ?;
