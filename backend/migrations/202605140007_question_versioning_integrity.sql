CREATE TABLE IF NOT EXISTS question_versions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  question_id uuid NOT NULL,
  version integer NOT NULL,
  actor_user_id uuid,
  event_type varchar(40) NOT NULL DEFAULT 'update',
  snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_question_versions_question_version
ON question_versions(question_id, version)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_question_versions_tenant_question
ON question_versions(tenant_id, question_id, version DESC)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_question_versions_snapshot
ON question_versions USING gin(snapshot)
WHERE deleted_at IS NULL;

UPDATE questions
SET version = COALESCE(version, 1)
WHERE version IS NULL OR version < 1;

INSERT INTO question_versions(id,tenant_id,question_id,version,actor_user_id,event_type,snapshot,metadata)
SELECT gen_random_uuid(),
       q.tenant_id,
       q.id,
       COALESCE(q.version, 1),
       q.owner_user_id,
       'backfill',
       jsonb_build_object(
         'id', q.id,
         'tenant_id', q.tenant_id,
         'code', q.code,
         'name', q.name,
         'description', COALESCE(q.description,''),
         'question_bank_id', q.question_bank_id,
         'lecturer_id', q.lecturer_id,
         'owner_user_id', q.owner_user_id,
         'question_type', COALESCE(q.question_type, 'multiple_choice'),
         'answer_mode', COALESCE(q.answer_mode, 'single'),
         'difficulty', COALESCE(q.difficulty, 'medium'),
         'content', COALESCE(q.content, ''),
         'explanation', COALESCE(q.explanation, ''),
         'score', CASE WHEN COALESCE(q.score, 0) > 0 THEN q.score ELSE 1 END,
         'status', q.status,
         'version', COALESCE(q.version, 1),
         'metadata', COALESCE(q.metadata, '{}'::jsonb),
         'options', COALESCE((
           SELECT jsonb_agg(
             jsonb_build_object(
               'id', qo.id,
               'label', qo.code,
               'text', COALESCE(qo.description,''),
               'is_correct', qo.is_correct,
               'sort_order', qo.sort_order,
               'metadata', COALESCE(qo.metadata,'{}'::jsonb)
             )
             ORDER BY qo.sort_order ASC, qo.created_at ASC
           )
           FROM question_options qo
           WHERE qo.question_id=q.id AND qo.tenant_id=q.tenant_id AND qo.deleted_at IS NULL
         ), '[]'::jsonb),
         'tags', COALESCE((
           SELECT jsonb_agg(
             jsonb_build_object(
               'id', qt.id,
               'name', qt.name,
               'owner_user_id', qt.owner_user_id,
               'lecturer_id', qt.lecturer_id
             )
             ORDER BY qt.name ASC
           )
           FROM question_tag_relations qtr
           JOIN question_tags qt ON qt.id=qtr.question_tag_id AND qt.deleted_at IS NULL
           WHERE qtr.question_id=q.id AND qtr.deleted_at IS NULL
         ), '[]'::jsonb),
         'media', COALESCE((
           SELECT jsonb_agg(
             jsonb_build_object(
               'id', qm.id,
               'option_id', qm.option_id,
               'usage_type', qm.usage_type,
               'object_key', qm.object_key,
               'mime_type', qm.mime_type,
               'file_size', qm.file_size,
               'sort_order', qm.sort_order
             )
             ORDER BY qm.usage_type ASC, qm.sort_order ASC, qm.created_at ASC
           )
           FROM question_media qm
           WHERE qm.question_id=q.id AND qm.tenant_id=q.tenant_id AND qm.deleted_at IS NULL
         ), '[]'::jsonb),
         'captured_at', now()
       ),
       jsonb_build_object('event_type','backfill','captured_at',now())
FROM questions q
WHERE q.tenant_id IS NOT NULL
ON CONFLICT (question_id, version) WHERE deleted_at IS NULL DO NOTHING;

INSERT INTO schema_migrations(version)
VALUES ('202605140007_question_versioning_integrity')
ON CONFLICT DO NOTHING;
