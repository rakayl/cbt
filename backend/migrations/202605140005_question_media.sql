CREATE TABLE IF NOT EXISTS question_media (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  question_id uuid NOT NULL,
  option_id uuid,
  media_type varchar(40) NOT NULL DEFAULT 'image',
  usage_type varchar(40) NOT NULL DEFAULT 'question',
  object_key text NOT NULL,
  original_filename text,
  mime_type varchar(120) NOT NULL,
  file_size bigint NOT NULL DEFAULT 0,
  width integer,
  height integer,
  sort_order integer NOT NULL DEFAULT 1,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_question_media_question ON question_media(tenant_id, question_id, usage_type, sort_order) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_question_media_option ON question_media(tenant_id, option_id, sort_order) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_question_media_object_key ON question_media(object_key) WHERE deleted_at IS NULL;

INSERT INTO schema_migrations(version)
VALUES ('202605140005_question_media')
ON CONFLICT DO NOTHING;
