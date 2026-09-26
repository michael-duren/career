-- Single-use record for MCP OAuth authorization codes and rotated refresh tokens.
-- Tokens are signed and stateless; this table only prevents replay across replicas.
CREATE TABLE oauth_consumed_tokens (
 jti TEXT PRIMARY KEY CHECK(length(jti) BETWEEN 16 AND 64),
 expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX oauth_consumed_tokens_expires_at ON oauth_consumed_tokens(expires_at);
