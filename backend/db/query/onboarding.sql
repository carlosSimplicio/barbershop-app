-- name: CreateRegistrationEstablishment :one
INSERT INTO establishments (name, slug, time_zone, created_at)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: CreateRegistrationOwner :one
INSERT INTO users (
    establishment_id, email, normalized_email, role, active, email_confirmed_at,
    credential_version, password_change_required, created_at, updated_at
)
VALUES ($1, $2, $3, $4, true, NULL, 1, false, $5, $5)
RETURNING id;

-- name: CreateRegistrationCredential :exec
INSERT INTO user_credentials (user_id, password_hash, created_at, updated_at)
VALUES ($1, $2, $3, $3);

-- name: CreateRegistrationEmailConfirmation :exec
INSERT INTO email_confirmations (user_id, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4);

-- name: InvalidateActiveEmailConfirmations :exec
UPDATE email_confirmations
SET invalidated_at = $2
WHERE user_id = $1
  AND consumed_at IS NULL
  AND invalidated_at IS NULL;

-- name: LockUserForEmailConfirmation :one
SELECT id
FROM users
WHERE id = $1
  AND role = 'owner'
  AND active = true
  AND email_confirmed_at IS NULL
FOR UPDATE;

-- name: FindUnconfirmedOwnerByNormalizedEmail :one
SELECT id, email
FROM users
WHERE normalized_email = $1
  AND role = 'owner'
  AND active = true
  AND email_confirmed_at IS NULL;

-- name: CreateEmailConfirmation :one
INSERT INTO email_confirmations (user_id, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, token_hash, expires_at, consumed_at, invalidated_at, created_at;

-- name: GetEmailConfirmationOwnerForUpdate :one
SELECT u.id
FROM users AS u
JOIN email_confirmations AS ec ON ec.user_id = u.id
WHERE ec.token_hash = $1
FOR UPDATE OF u;

-- name: GetEmailConfirmationForUpdate :one
SELECT
    ec.id,
    ec.user_id,
    ec.token_hash,
    ec.expires_at,
    ec.consumed_at,
    ec.invalidated_at,
    ec.created_at,
    u.establishment_id,
    u.email,
    u.role,
    u.active,
    u.email_confirmed_at,
    u.credential_version,
    u.password_change_required
FROM email_confirmations AS ec
JOIN users AS u ON u.id = ec.user_id
WHERE ec.token_hash = $1
FOR UPDATE OF ec;

-- name: ConsumeEmailConfirmation :execrows
UPDATE email_confirmations
SET consumed_at = $2
WHERE id = $1
  AND consumed_at IS NULL
  AND invalidated_at IS NULL;

-- name: ConfirmUserEmail :execrows
UPDATE users
SET email_confirmed_at = $2,
    updated_at = $2
WHERE id = $1
  AND email_confirmed_at IS NULL;

-- name: CreateEstablishmentTrial :one
INSERT INTO establishment_trials (owner_user_id, establishment_id, starts_at, ends_at, created_at)
SELECT
    u.id,
    u.establishment_id,
    sqlc.arg(starts_at)::timestamptz,
    sqlc.arg(ends_at)::timestamptz,
    sqlc.arg(starts_at)::timestamptz
FROM users AS u
WHERE u.id = sqlc.arg(owner_user_id)
  AND u.establishment_id = sqlc.arg(establishment_id)
  AND u.role = 'owner'
RETURNING id, owner_user_id, establishment_id, starts_at, ends_at, created_at;

-- name: GetEstablishmentWithTrial :one
SELECT e.id, e.name, e.slug, e.time_zone, t.starts_at, t.ends_at
FROM establishments AS e
LEFT JOIN establishment_trials AS t ON t.establishment_id = e.id
WHERE e.id = $1;

-- name: GetEstablishmentPublicSettings :one
SELECT
    id,
    description,
    address,
    contact_phone,
    minimum_booking_notice_minutes,
    booking_horizon_days,
    customer_change_deadline_minutes
FROM establishments
WHERE id = $1;

-- name: GetEstablishmentSettingsByID :one
SELECT
    e.id, e.name, e.slug, e.time_zone, e.description, e.address, e.contact_phone,
    e.minimum_booking_notice_minutes, e.booking_horizon_days, e.customer_change_deadline_minutes,
    e.operational_data_started_at,
    l.content_type AS logo_content_type, l.byte_size AS logo_byte_size,
    l.width AS logo_width, l.height AS logo_height,
    COALESCE((
        SELECT jsonb_agg(jsonb_build_object('label', links.label, 'url', links.url) ORDER BY links.position)
        FROM establishment_social_links AS links
        WHERE links.establishment_id = e.id
    ), '[]'::jsonb)::text AS social_links
FROM establishments AS e
LEFT JOIN establishment_logos AS l ON l.establishment_id = e.id AND l.id = e.logo_id
WHERE e.id = $1;

-- name: LockEstablishmentOperationalWrite :one
SELECT time_zone, operational_data_started_at
FROM establishments
WHERE id = $1
FOR UPDATE;

-- name: MarkEstablishmentOperationalDataStarted :execrows
UPDATE establishments
SET operational_data_started_at = COALESCE(operational_data_started_at, $2, statement_timestamp())
WHERE id = $1;

-- name: ListEstablishmentSocialLinks :many
SELECT establishment_id, position, label, url
FROM establishment_social_links
WHERE establishment_id = $1
ORDER BY position;

-- name: UpdateEstablishmentPublicSettings :execrows
UPDATE establishments
SET description = sqlc.narg('description'),
    address = sqlc.narg('address'),
    contact_phone = sqlc.narg('contact_phone'),
    minimum_booking_notice_minutes = sqlc.narg('minimum_booking_notice_minutes'),
    booking_horizon_days = sqlc.narg('booking_horizon_days'),
    customer_change_deadline_minutes = sqlc.narg('customer_change_deadline_minutes')
WHERE id = sqlc.arg('establishment_id');

-- name: PatchEstablishmentSettings :execrows
UPDATE establishments
SET name = CASE WHEN sqlc.arg('name_set')::boolean THEN sqlc.arg('name')::text ELSE name END,
    time_zone = CASE WHEN sqlc.arg('time_zone_set')::boolean THEN sqlc.arg('time_zone')::text ELSE time_zone END,
    description = CASE WHEN sqlc.arg('description_set')::boolean THEN sqlc.narg('description')::text ELSE description END,
    address = CASE WHEN sqlc.arg('address_set')::boolean THEN sqlc.narg('address')::text ELSE address END,
    contact_phone = CASE WHEN sqlc.arg('contact_phone_set')::boolean THEN sqlc.narg('contact_phone')::text ELSE contact_phone END,
    minimum_booking_notice_minutes = CASE WHEN sqlc.arg('minimum_booking_notice_minutes_set')::boolean THEN sqlc.narg('minimum_booking_notice_minutes')::integer ELSE minimum_booking_notice_minutes END,
    booking_horizon_days = CASE WHEN sqlc.arg('booking_horizon_days_set')::boolean THEN sqlc.narg('booking_horizon_days')::integer ELSE booking_horizon_days END,
    customer_change_deadline_minutes = CASE WHEN sqlc.arg('customer_change_deadline_minutes_set')::boolean THEN sqlc.narg('customer_change_deadline_minutes')::integer ELSE customer_change_deadline_minutes END
WHERE id = sqlc.arg('establishment_id');

-- name: DeleteEstablishmentSocialLinks :exec
DELETE FROM establishment_social_links
WHERE establishment_id = $1;

-- name: CreateEstablishmentSocialLink :exec
INSERT INTO establishment_social_links (establishment_id, position, label, url)
VALUES ($1, $2, $3, $4);

-- name: CreateEstablishmentLogo :one
INSERT INTO establishment_logos (
    establishment_id, object_key, content_type, byte_size, width, height, content_sha256
)
VALUES (
    sqlc.arg('establishment_id'), sqlc.arg('object_key'), sqlc.arg('content_type'),
    sqlc.arg('byte_size'), sqlc.arg('width'), sqlc.arg('height'), sqlc.arg('content_sha256')
)
RETURNING id, establishment_id, object_key, content_type, byte_size, width, height, content_sha256, created_at;

-- name: SetCurrentEstablishmentLogo :execrows
UPDATE establishments
SET logo_id = sqlc.arg('logo_id')
WHERE id = sqlc.arg('establishment_id')
  AND sqlc.arg('logo_id')::uuid IS NOT NULL;

-- name: GetCurrentEstablishmentLogo :one
SELECT
    l.id, l.establishment_id, l.object_key, l.content_type, l.byte_size,
    l.width, l.height, l.content_sha256, l.created_at
FROM establishments AS e
JOIN establishment_logos AS l
    ON l.establishment_id = e.id AND l.id = e.logo_id
WHERE e.id = $1;

-- name: GetPublicEstablishmentBySlug :one
SELECT id, name, slug
FROM establishments
WHERE slug = $1
  AND logo_id IS NOT NULL
  AND btrim(name) <> '';

-- name: GetPublicEstablishmentProfile :one
SELECT
    e.name,
    e.slug,
    e.description,
    e.address,
    e.contact_phone,
    COALESCE(
        jsonb_agg(jsonb_build_object('label', links.label, 'url', links.url) ORDER BY links.position)
            FILTER (WHERE links.position IS NOT NULL),
        '[]'::jsonb
    )::text AS social_links
FROM establishments AS e
JOIN establishment_logos AS logo
    ON logo.establishment_id = e.id AND logo.id = e.logo_id
LEFT JOIN establishment_social_links AS links
    ON links.establishment_id = e.id
WHERE e.id = sqlc.arg('establishment_id')
  AND e.slug = sqlc.arg('slug')
  AND btrim(e.name) <> ''
GROUP BY e.id;

-- name: GetPublicEstablishmentLogo :one
SELECT l.object_key, l.content_type, l.byte_size, l.content_sha256
FROM establishments AS e
JOIN establishment_logos AS l
    ON l.establishment_id = e.id AND l.id = e.logo_id
WHERE e.id = sqlc.arg('establishment_id')
  AND e.slug = sqlc.arg('slug');
