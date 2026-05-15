DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relname='proctoring_logs' AND relkind <> 'p') THEN
    ALTER TABLE proctoring_logs RENAME TO proctoring_logs_legacy;
    CREATE TABLE proctoring_logs (
      id uuid DEFAULT gen_random_uuid(),
      tenant_id uuid NULL,
      code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
      name varchar(160) NOT NULL DEFAULT 'proctoring_logs',
      description text NULL,
      status varchar(40) NOT NULL DEFAULT 'active',
      metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
      created_at timestamptz NOT NULL DEFAULT now(),
      updated_at timestamptz NOT NULL DEFAULT now(),
      deleted_at timestamptz NULL,
      exam_session_id uuid,
      event_type varchar(80),
      score numeric(6,2) NOT NULL DEFAULT 0
    ) PARTITION BY RANGE (created_at);
    INSERT INTO proctoring_logs SELECT * FROM proctoring_logs_legacy;
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS proctoring_logs_2026 PARTITION OF proctoring_logs
  FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');
CREATE TABLE IF NOT EXISTS proctoring_logs_default PARTITION OF proctoring_logs DEFAULT;
CREATE INDEX IF NOT EXISTS idx_proctoring_logs_2026_tenant_created ON proctoring_logs_2026(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_proctoring_logs_2026_session ON proctoring_logs_2026(exam_session_id);

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relname='browser_activity_logs' AND relkind <> 'p') THEN
    ALTER TABLE browser_activity_logs RENAME TO browser_activity_logs_legacy;
    CREATE TABLE browser_activity_logs (
      id uuid DEFAULT gen_random_uuid(),
      tenant_id uuid NULL,
      code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
      name varchar(160) NOT NULL DEFAULT 'browser_activity_logs',
      description text NULL,
      status varchar(40) NOT NULL DEFAULT 'active',
      metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
      created_at timestamptz NOT NULL DEFAULT now(),
      updated_at timestamptz NOT NULL DEFAULT now(),
      deleted_at timestamptz NULL,
      exam_session_id uuid,
      activity_type varchar(80)
    ) PARTITION BY RANGE (created_at);
    INSERT INTO browser_activity_logs SELECT * FROM browser_activity_logs_legacy;
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS browser_activity_logs_2026 PARTITION OF browser_activity_logs
  FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');
CREATE TABLE IF NOT EXISTS browser_activity_logs_default PARTITION OF browser_activity_logs DEFAULT;
CREATE INDEX IF NOT EXISTS idx_browser_activity_logs_2026_tenant_created ON browser_activity_logs_2026(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_browser_activity_logs_2026_session ON browser_activity_logs_2026(exam_session_id);

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relname='face_detection_logs' AND relkind <> 'p') THEN
    ALTER TABLE face_detection_logs RENAME TO face_detection_logs_legacy;
    CREATE TABLE face_detection_logs (
      id uuid DEFAULT gen_random_uuid(),
      tenant_id uuid NULL,
      code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
      name varchar(160) NOT NULL DEFAULT 'face_detection_logs',
      description text NULL,
      status varchar(40) NOT NULL DEFAULT 'active',
      metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
      created_at timestamptz NOT NULL DEFAULT now(),
      updated_at timestamptz NOT NULL DEFAULT now(),
      deleted_at timestamptz NULL,
      exam_session_id uuid,
      face_count integer NOT NULL DEFAULT 0
    ) PARTITION BY RANGE (created_at);
    INSERT INTO face_detection_logs SELECT * FROM face_detection_logs_legacy;
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS face_detection_logs_2026 PARTITION OF face_detection_logs
  FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');
CREATE TABLE IF NOT EXISTS face_detection_logs_default PARTITION OF face_detection_logs DEFAULT;
CREATE INDEX IF NOT EXISTS idx_face_detection_logs_2026_tenant_created ON face_detection_logs_2026(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_face_detection_logs_2026_session ON face_detection_logs_2026(exam_session_id);

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relname='recovery_logs' AND relkind <> 'p') THEN
    ALTER TABLE recovery_logs RENAME TO recovery_logs_legacy;
    CREATE TABLE recovery_logs (
      id uuid DEFAULT gen_random_uuid(),
      tenant_id uuid NULL,
      code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
      name varchar(160) NOT NULL DEFAULT 'recovery_logs',
      description text NULL,
      status varchar(40) NOT NULL DEFAULT 'active',
      metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
      created_at timestamptz NOT NULL DEFAULT now(),
      updated_at timestamptz NOT NULL DEFAULT now(),
      deleted_at timestamptz NULL,
      exam_session_id uuid,
      event_type varchar(80),
      client_time timestamptz
    ) PARTITION BY RANGE (created_at);
    INSERT INTO recovery_logs SELECT * FROM recovery_logs_legacy;
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS recovery_logs_2026 PARTITION OF recovery_logs
  FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');
CREATE TABLE IF NOT EXISTS recovery_logs_default PARTITION OF recovery_logs DEFAULT;
CREATE INDEX IF NOT EXISTS idx_recovery_logs_2026_tenant_created ON recovery_logs_2026(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recovery_logs_2026_session ON recovery_logs_2026(exam_session_id);

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relname='reconnect_logs' AND relkind <> 'p') THEN
    ALTER TABLE reconnect_logs RENAME TO reconnect_logs_legacy;
    CREATE TABLE reconnect_logs (
      id uuid DEFAULT gen_random_uuid(),
      tenant_id uuid NULL,
      code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
      name varchar(160) NOT NULL DEFAULT 'reconnect_logs',
      description text NULL,
      status varchar(40) NOT NULL DEFAULT 'active',
      metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
      created_at timestamptz NOT NULL DEFAULT now(),
      updated_at timestamptz NOT NULL DEFAULT now(),
      deleted_at timestamptz NULL,
      exam_session_id uuid,
      disconnected_at timestamptz,
      reconnected_at timestamptz,
      auto_submitted boolean NOT NULL DEFAULT false
    ) PARTITION BY RANGE (created_at);
    INSERT INTO reconnect_logs SELECT * FROM reconnect_logs_legacy;
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS reconnect_logs_2026 PARTITION OF reconnect_logs
  FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');
CREATE TABLE IF NOT EXISTS reconnect_logs_default PARTITION OF reconnect_logs DEFAULT;
CREATE INDEX IF NOT EXISTS idx_reconnect_logs_2026_tenant_created ON reconnect_logs_2026(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_reconnect_logs_2026_session ON reconnect_logs_2026(exam_session_id);

INSERT INTO schema_migrations(version)
VALUES ('202605130004_partition_high_volume_events')
ON CONFLICT DO NOTHING;
