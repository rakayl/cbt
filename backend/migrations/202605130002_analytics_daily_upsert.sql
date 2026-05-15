CREATE UNIQUE INDEX IF NOT EXISTS uq_analytics_daily_tenant_date ON analytics_daily(tenant_id, date);

INSERT INTO schema_migrations(version)
VALUES ('202605130002_analytics_daily_upsert')
ON CONFLICT DO NOTHING;
