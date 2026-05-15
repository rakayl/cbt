ALTER TABLE question_banks ADD COLUMN IF NOT EXISTS lecturer_id uuid;
ALTER TABLE question_banks ADD COLUMN IF NOT EXISTS owner_user_id uuid;

CREATE INDEX IF NOT EXISTS idx_question_banks_lecturer
ON question_banks(tenant_id, lecturer_id)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_question_banks_owner_user
ON question_banks(tenant_id, owner_user_id)
WHERE deleted_at IS NULL;

WITH owners AS (
  SELECT DISTINCT ON (question_bank_id)
    question_bank_id,
    lecturer_id,
    owner_user_id
  FROM questions
  WHERE deleted_at IS NULL
    AND question_bank_id IS NOT NULL
    AND (lecturer_id IS NOT NULL OR owner_user_id IS NOT NULL)
  ORDER BY question_bank_id, updated_at DESC NULLS LAST, created_at DESC NULLS LAST
)
UPDATE question_banks qb
SET lecturer_id = COALESCE(qb.lecturer_id, owners.lecturer_id),
    owner_user_id = COALESCE(qb.owner_user_id, owners.owner_user_id),
    metadata = COALESCE(qb.metadata, '{}'::jsonb) || jsonb_strip_nulls(jsonb_build_object(
      'lecturer_id', COALESCE(qb.lecturer_id, owners.lecturer_id),
      'owner_user_id', COALESCE(qb.owner_user_id, owners.owner_user_id)
    ))
FROM owners
WHERE qb.id = owners.question_bank_id
  AND qb.deleted_at IS NULL
  AND (qb.lecturer_id IS NULL OR qb.owner_user_id IS NULL);

INSERT INTO schema_migrations(version)
VALUES ('202605150001_question_bank_teacher_ownership')
ON CONFLICT DO NOTHING;
