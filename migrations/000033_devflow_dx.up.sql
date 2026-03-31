ALTER TABLE ext_environments ADD COLUMN IF NOT EXISTS compose_status VARCHAR(20) NOT NULL DEFAULT 'stopped';
ALTER TABLE ext_environments ADD COLUMN IF NOT EXISTS code_server_port INT;
ALTER TABLE ext_environments ADD COLUMN IF NOT EXISTS preview_port INT;

COMMENT ON COLUMN ext_environments.compose_status IS 'stopped | starting | running | error';
