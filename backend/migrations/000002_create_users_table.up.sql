-- Reusable trigger function: keeps updated_at current on every UPDATE.
-- Every table with an updated_at column from here on attaches this same
-- trigger instead of redefining the logic per table.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Dashboard/system accounts (Admin, HR, Management, ...). Distinct from
-- `employees` (Phase 3), which represents people whose attendance is
-- tracked and who never log into the dashboard themselves.
--
-- `role` is a plain VARCHAR with a CHECK constraint rather than a native
-- Postgres ENUM: it gives the same data-integrity guarantee (only these
-- four values are ever accepted) without the extra Go-driver ceremony of
-- registering a custom enum type with pgx, and adding a new role later is
-- a plain ALTER TABLE ... CHECK instead of an ALTER TYPE migration.
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(150) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(20) NOT NULL
                  CHECK (role IN ('SUPER_ADMIN', 'ADMIN', 'HR', 'MANAGEMENT')),
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
