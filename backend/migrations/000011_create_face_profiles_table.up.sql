-- One face profile per employee. feature_vector holds whatever numeric
-- representation the active FaceRecognitionService implementation
-- produces (Phase 5's tablet app: a geometric landmark-distance vector;
-- a future deep-learning embedding would fit the same JSONB column
-- without a schema change). Stored as JSONB rather than a Postgres array
-- type so the vector's length/meaning can evolve with the recognition
-- engine without a migration.
CREATE TABLE face_profiles (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id  UUID NOT NULL UNIQUE REFERENCES employees(id) ON DELETE CASCADE,
    feature_vector JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_face_profiles_set_updated_at
    BEFORE UPDATE ON face_profiles
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
