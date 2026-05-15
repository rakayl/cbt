-- Exam session questions store immutable question snapshots in metadata->'snapshot'.
-- The snapshot is written when a session starts so historical exams still render
-- and grade correctly even if the original bank question, options, or media are
-- edited or soft-deleted later.
--
-- Backfill existing session questions with the current bank-soal state. This
-- cannot recover older text if a question was already edited before this
-- migration, but it protects every existing session from future edits/deletes.
UPDATE exam_session_questions esq
SET metadata = COALESCE(esq.metadata, '{}'::jsonb) || jsonb_build_object(
  'snapshot_version', 1,
  'snapshot', jsonb_build_object(
    'id', q.id,
    'question_id', q.id,
    'text', COALESCE(q.content, q.name, ''),
    'question_type', COALESCE(q.question_type, 'multiple_choice'),
    'answer_mode', COALESCE(q.answer_mode, 'single'),
    'score', CASE WHEN COALESCE(q.score, 0) > 0 THEN q.score ELSE 1 END,
    'captured_at', now(),
    'media', COALESCE((
      SELECT jsonb_agg(
        jsonb_build_object(
          'id', qm.id,
          'mime_type', qm.mime_type,
          'file_size', qm.file_size,
          'sort_order', qm.sort_order
        )
        ORDER BY qm.sort_order ASC, qm.created_at ASC
      )
      FROM question_media qm
      WHERE qm.tenant_id IS NOT DISTINCT FROM esq.tenant_id
        AND qm.question_id = q.id
        AND qm.option_id IS NULL
        AND qm.usage_type = 'question'
        AND qm.deleted_at IS NULL
    ), '[]'::jsonb),
    'options', COALESCE((
      SELECT jsonb_agg(
        jsonb_build_object(
          'id', qo.id,
          'label', qo.code,
          'text', COALESCE(qo.description, ''),
          'is_correct', qo.is_correct,
          'sort_order', qo.sort_order,
          'media', COALESCE((
            SELECT jsonb_agg(
              jsonb_build_object(
                'id', qmo.id,
                'mime_type', qmo.mime_type,
                'file_size', qmo.file_size,
                'sort_order', qmo.sort_order
              )
              ORDER BY qmo.sort_order ASC, qmo.created_at ASC
            )
            FROM question_media qmo
            WHERE qmo.tenant_id IS NOT DISTINCT FROM esq.tenant_id
              AND qmo.question_id = q.id
              AND qmo.option_id = qo.id
              AND qmo.usage_type = 'option'
              AND qmo.deleted_at IS NULL
          ), '[]'::jsonb)
        )
        ORDER BY qo.sort_order ASC, qo.created_at ASC
      )
      FROM question_options qo
      WHERE qo.tenant_id IS NOT DISTINCT FROM esq.tenant_id
        AND qo.question_id = q.id
        AND qo.deleted_at IS NULL
    ), '[]'::jsonb)
  )
)
FROM questions q
WHERE q.id = esq.question_id
  AND q.tenant_id IS NOT DISTINCT FROM esq.tenant_id
  AND esq.deleted_at IS NULL
  AND NOT (COALESCE(esq.metadata, '{}'::jsonb) ? 'snapshot');

CREATE INDEX IF NOT EXISTS idx_exam_session_questions_snapshot
ON exam_session_questions USING gin ((metadata->'snapshot'))
WHERE deleted_at IS NULL;

INSERT INTO schema_migrations(version)
VALUES ('202605140006_exam_question_snapshots')
ON CONFLICT DO NOTHING;
