WITH perms(code,name,resource,action) AS (
  VALUES
    ('users:read','Users Read','users','read'),
    ('users:write','Users Write','users','write')
)
INSERT INTO permissions(id,tenant_id,code,name,resource,action,status,metadata)
SELECT gen_random_uuid(), t.id, p.code, p.name, p.resource, p.action, 'active', '{"seed":true}'::jsonb
FROM tenants t
CROSS JOIN perms p
WHERE NOT EXISTS (
  SELECT 1 FROM permissions existing
  WHERE existing.tenant_id=t.id AND existing.code=p.code AND existing.deleted_at IS NULL
);

INSERT INTO role_permissions(id,tenant_id,code,name,role_id,permission_id,status,metadata)
SELECT gen_random_uuid(), r.tenant_id, r.code || '_' || p.code, r.name || ' ' || p.name, r.id, p.id, 'active', '{"seed":true}'::jsonb
FROM roles r
JOIN permissions p ON p.tenant_id=r.tenant_id AND p.deleted_at IS NULL
WHERE r.code='SUPER_ADMIN'
  AND p.code IN ('users:read','users:write')
ON CONFLICT (role_id, permission_id) WHERE deleted_at IS NULL DO NOTHING;

INSERT INTO schema_migrations(version)
VALUES ('202605130007_super_admin_user_management')
ON CONFLICT DO NOTHING;
