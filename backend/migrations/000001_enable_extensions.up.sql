-- Enable pgcrypto for gen_random_uuid(), used as the default for every
-- entity's primary key from Phase 3 onward (employees, departments,
-- attendances, ...). Enabling it here in Phase 1 keeps schema migrations in
-- later phases free of setup concerns.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
