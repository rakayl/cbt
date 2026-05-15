ALTER TABLE questions ADD COLUMN IF NOT EXISTS lecturer_id uuid;
ALTER TABLE questions ADD COLUMN IF NOT EXISTS owner_user_id uuid;

CREATE INDEX IF NOT EXISTS idx_questions_lecturer ON questions(tenant_id, lecturer_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_questions_owner_user ON questions(tenant_id, owner_user_id) WHERE deleted_at IS NULL;

UPDATE questions
SET lecturer_id = NULLIF(metadata->>'lecturer_id', '')::uuid
WHERE lecturer_id IS NULL
  AND metadata ? 'lecturer_id'
  AND NULLIF(metadata->>'lecturer_id', '') IS NOT NULL;

UPDATE questions
SET owner_user_id = NULLIF(metadata->>'owner_user_id', '')::uuid
WHERE owner_user_id IS NULL
  AND metadata ? 'owner_user_id'
  AND NULLIF(metadata->>'owner_user_id', '') IS NOT NULL;

INSERT INTO schema_migrations(version)
VALUES ('202605130009_question_teacher_ownership')
ON CONFLICT DO NOTHING;
