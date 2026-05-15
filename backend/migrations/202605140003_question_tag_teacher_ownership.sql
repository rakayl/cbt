ALTER TABLE question_tags ADD COLUMN IF NOT EXISTS lecturer_id uuid;
ALTER TABLE question_tags ADD COLUMN IF NOT EXISTS owner_user_id uuid;

CREATE INDEX IF NOT EXISTS idx_question_tags_lecturer
ON question_tags(tenant_id, lecturer_id)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_question_tags_owner_user
ON question_tags(tenant_id, owner_user_id)
WHERE deleted_at IS NULL;

WITH metadata_owner AS (
  SELECT
    id,
    NULLIF(metadata->>'lecturer_id', '')::uuid AS lecturer_id,
    NULLIF(metadata->>'owner_user_id', '')::uuid AS owner_user_id
  FROM question_tags
  WHERE deleted_at IS NULL
    AND (metadata ? 'lecturer_id' OR metadata ? 'owner_user_id')
),
relation_owner AS (
  SELECT DISTINCT ON (qtr.question_tag_id)
    qtr.question_tag_id AS id,
    q.lecturer_id,
    q.owner_user_id
  FROM question_tag_relations qtr
  JOIN questions q ON q.id=qtr.question_id AND q.deleted_at IS NULL
  WHERE qtr.deleted_at IS NULL
    AND (q.lecturer_id IS NOT NULL OR q.owner_user_id IS NOT NULL)
  ORDER BY qtr.question_tag_id, q.created_at DESC
),
resolved_owner AS (
  SELECT
    qt.id,
    COALESCE(qt.lecturer_id, mo.lecturer_id, ro.lecturer_id) AS lecturer_id,
    COALESCE(qt.owner_user_id, mo.owner_user_id, ro.owner_user_id) AS owner_user_id
  FROM question_tags qt
  LEFT JOIN metadata_owner mo ON mo.id=qt.id
  LEFT JOIN relation_owner ro ON ro.id=qt.id
  WHERE qt.deleted_at IS NULL
)
UPDATE question_tags qt
SET lecturer_id=ro.lecturer_id,
    owner_user_id=ro.owner_user_id,
    metadata=COALESCE(qt.metadata, '{}'::jsonb)
      || jsonb_strip_nulls(jsonb_build_object(
        'lecturer_id', ro.lecturer_id::text,
        'owner_user_id', ro.owner_user_id::text
      )),
    updated_at=now()
FROM resolved_owner ro
WHERE qt.id=ro.id
  AND (ro.lecturer_id IS NOT NULL OR ro.owner_user_id IS NOT NULL);

INSERT INTO schema_migrations(version)
VALUES ('202605140003_question_tag_teacher_ownership')
ON CONFLICT DO NOTHING;
