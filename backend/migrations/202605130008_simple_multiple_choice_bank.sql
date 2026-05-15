ALTER TABLE questions ADD COLUMN IF NOT EXISTS answer_mode varchar(20) NOT NULL DEFAULT 'single';
ALTER TABLE questions ADD COLUMN IF NOT EXISTS score numeric(8,2) NOT NULL DEFAULT 1;
ALTER TABLE question_options ADD COLUMN IF NOT EXISTS sort_order integer NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_questions_bank_mode ON questions(tenant_id, question_bank_id, answer_mode) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_question_options_question_order ON question_options(question_id, sort_order) WHERE deleted_at IS NULL;

INSERT INTO schema_migrations(version)
VALUES ('202605130008_simple_multiple_choice_bank')
ON CONFLICT DO NOTHING;
