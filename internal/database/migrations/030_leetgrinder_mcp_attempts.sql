-- Attempts logged from chat through the MCP log_leetgrinder_attempt tool
-- record source 'mcp'.
ALTER TABLE leetgrinder_attempts DROP CONSTRAINT leetgrinder_attempts_source_check;
ALTER TABLE leetgrinder_attempts ADD CONSTRAINT leetgrinder_attempts_source_check
    CHECK (source IN ('web', 'extension', 'mcp'));
