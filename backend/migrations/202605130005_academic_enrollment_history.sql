ALTER TABLE class_rooms ADD COLUMN IF NOT EXISTS study_program_id uuid;
ALTER TABLE class_rooms ADD COLUMN IF NOT EXISTS academic_period_id uuid;
ALTER TABLE class_rooms ADD COLUMN IF NOT EXISTS capacity integer;

ALTER TABLE enrollment ADD COLUMN IF NOT EXISTS student_id uuid;
ALTER TABLE enrollment ADD COLUMN IF NOT EXISTS class_room_id uuid;
ALTER TABLE enrollment ADD COLUMN IF NOT EXISTS study_program_id uuid;
ALTER TABLE enrollment ADD COLUMN IF NOT EXISTS enrolled_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE enrollment ADD COLUMN IF NOT EXISTS exited_at timestamptz;
ALTER TABLE enrollment ADD COLUMN IF NOT EXISTS active boolean NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_class_rooms_study_program ON class_rooms(tenant_id, study_program_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_enrollment_student_history ON enrollment(tenant_id, student_id, enrolled_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_enrollment_class_active ON enrollment(tenant_id, class_room_id, active) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_enrollment_active_student ON enrollment(tenant_id, student_id) WHERE deleted_at IS NULL AND active = true;

INSERT INTO schema_migrations(version)
VALUES ('202605130005_academic_enrollment_history')
ON CONFLICT DO NOTHING;
