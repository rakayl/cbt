WITH default_plan AS (
  SELECT id FROM subscription_plans WHERE code='ENT' LIMIT 1
),
seed_tenant AS (
  INSERT INTO tenants(id,code,name,status,domain,subscription_plan_id,metadata)
  VALUES(
    '10000000-0000-0000-0000-000000000001',
    'DEFAULT',
    'Default Campus',
    'active',
    'default.local',
    (SELECT id FROM default_plan),
    '{"seed":true}'::jsonb
  )
  ON CONFLICT (id) DO UPDATE
  SET subscription_plan_id=excluded.subscription_plan_id, updated_at=now()
  RETURNING id
),
seed_user AS (
  INSERT INTO users(id,tenant_id,code,name,email,password_hash,status,metadata)
  VALUES(
    '20000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    'ADMIN',
    'Default Administrator',
    'admin@example.edu',
    crypt('ChangeMe123!change-me-password-pepper', gen_salt('bf', 12)),
    'active',
    '{"seed":true}'::jsonb
  )
  ON CONFLICT (tenant_id, email) WHERE deleted_at IS NULL DO UPDATE
  SET password_hash=excluded.password_hash, updated_at=now()
  RETURNING id
),
seed_role AS (
  INSERT INTO roles(id,tenant_id,code,name,status,metadata)
  VALUES(
    '30000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    'SUPER_ADMIN',
    'Super Admin',
    'active',
    '{"seed":true}'::jsonb
  )
  ON CONFLICT (id) DO UPDATE SET updated_at=now()
  RETURNING id
),
seed_permission AS (
  INSERT INTO permissions(id,tenant_id,code,name,resource,action,status,metadata)
  VALUES(
    '40000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    '*',
    'All Permissions',
    '*',
    '*',
    'active',
    '{"seed":true}'::jsonb
  )
  ON CONFLICT (id) DO UPDATE SET updated_at=now()
  RETURNING id
)
INSERT INTO user_roles(id,tenant_id,code,name,user_id,role_id,status,metadata)
VALUES(
  '50000000-0000-0000-0000-000000000001',
  '10000000-0000-0000-0000-000000000001',
  'ADMIN_SUPER_ADMIN',
  'Admin Super Admin',
  '20000000-0000-0000-0000-000000000001',
  '30000000-0000-0000-0000-000000000001',
  'active',
  '{"seed":true}'::jsonb
)
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions(id,tenant_id,code,name,role_id,permission_id,status,metadata)
VALUES(
  '60000000-0000-0000-0000-000000000001',
  '10000000-0000-0000-0000-000000000001',
  'SUPER_ADMIN_ALL',
  'Super Admin All Permissions',
  '30000000-0000-0000-0000-000000000001',
  '40000000-0000-0000-0000-000000000001',
  'active',
  '{"seed":true}'::jsonb
)
ON CONFLICT DO NOTHING;

INSERT INTO schema_migrations(version)
VALUES ('202605130003_seed_default_admin')
ON CONFLICT DO NOTHING;
