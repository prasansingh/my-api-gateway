CREATE TABLE IF NOT EXISTS api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash    VARCHAR(64) NOT NULL UNIQUE,
    key_prefix  VARCHAR(8) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
