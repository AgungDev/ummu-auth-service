CREATE EXTENSION IF NOT EXISTS "pgcrypto";

ALTER TABLE user_roles
    ADD COLUMN IF NOT EXISTS scope_type VARCHAR(32) NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS scope_id VARCHAR(128) NOT NULL DEFAULT '';

DO $$
DECLARE
    constraint_name TEXT;
BEGIN
    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'user_roles'::regclass
      AND contype = 'p';

    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE user_roles DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

ALTER TABLE user_roles
    ADD CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role_id, scope_type, scope_id);

UPDATE user_roles assignments
SET scope_type = 'global', scope_id = ''
FROM roles
WHERE assignments.role_id = roles.id
  AND roles.code = 'super_admin'
  AND assignments.scope_type = 'none';

UPDATE refresh_tokens
SET refresh_token_hash = encode(digest(refresh_token_hash, 'sha256'), 'hex')
WHERE length(refresh_token_hash) <> 64;

CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_hash
ON refresh_tokens(refresh_token_hash);

CREATE TABLE IF NOT EXISTS client_scopes (
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    scope VARCHAR(128) NOT NULL,
    PRIMARY KEY (client_id, scope)
);

CREATE TABLE IF NOT EXISTS client_audiences (
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    audience VARCHAR(255) NOT NULL,
    PRIMARY KEY (client_id, audience)
);

CREATE TABLE IF NOT EXISTS client_access_contexts (
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    scope_type VARCHAR(32) NOT NULL,
    scope_id VARCHAR(128) NOT NULL DEFAULT '',
    PRIMARY KEY (client_id, scope_type, scope_id)
);

CREATE INDEX IF NOT EXISTS idx_client_scopes_client ON client_scopes(client_id);
CREATE INDEX IF NOT EXISTS idx_client_audiences_client ON client_audiences(client_id);
