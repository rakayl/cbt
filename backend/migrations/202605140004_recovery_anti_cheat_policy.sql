UPDATE exams
SET metadata = COALESCE(metadata, '{}'::jsonb)
  || jsonb_build_object(
    'recovery_policy', COALESCE(metadata->'recovery_policy', '{
      "timer_mode":"recovery_pause",
      "max_pause_seconds":3600,
      "max_reconnect_attempts":3,
      "device_change_requires_approval":true,
      "auto_submit_when_recovery_exceeded":false
    }'::jsonb),
    'anti_cheat_policy', COALESCE(metadata->'anti_cheat_policy', '{
      "fullscreen_required":true,
      "webcam_required":true,
      "block_copy_paste":true,
      "block_right_click":true,
      "snapshot_interval_seconds":60,
      "critical_score_threshold":90
    }'::jsonb)
  )
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_exam_sessions_recovery_paused
ON exam_sessions(tenant_id, status_enum)
WHERE deleted_at IS NULL
  AND status_enum='reconnecting'
  AND metadata->'recovery'->>'timer_paused'='true';

CREATE INDEX IF NOT EXISTS idx_proctoring_logs_session_event
ON proctoring_logs(exam_session_id, event_type, created_at DESC);

INSERT INTO schema_migrations(version)
VALUES ('202605140004_recovery_anti_cheat_policy')
ON CONFLICT DO NOTHING;
