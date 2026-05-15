INSERT INTO permissions(id,tenant_id,code,name,resource,action,status,metadata)
SELECT gen_random_uuid(), '10000000-0000-0000-0000-000000000001', item.code, item.name, item.resource, item.action, 'active', '{}'::jsonb
FROM (
  VALUES
    ('roles:read', 'Roles Read', 'roles', 'read'),
    ('roles:write', 'Roles Write', 'roles', 'write'),
    ('permissions:read', 'Permissions Read', 'permissions', 'read'),
    ('permissions:write', 'Permissions Write', 'permissions', 'write')
) AS item(code,name,resource,action)
WHERE NOT EXISTS (
  SELECT 1
  FROM permissions p
  WHERE p.tenant_id='10000000-0000-0000-0000-000000000001'
    AND p.code=item.code
    AND p.deleted_at IS NULL
);

INSERT INTO role_permissions(id,tenant_id,code,name,role_id,permission_id,status,metadata)
SELECT gen_random_uuid(), r.tenant_id, r.code || '_' || p.code, r.name || ' ' || p.name, r.id, p.id, 'active', '{}'::jsonb
FROM roles r
JOIN permissions p ON p.tenant_id=r.tenant_id AND p.code IN ('roles:read','roles:write','permissions:read','permissions:write') AND p.deleted_at IS NULL
WHERE r.code='SUPER_ADMIN'
  AND r.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM role_permissions rp
    WHERE rp.role_id=r.id AND rp.permission_id=p.id AND rp.deleted_at IS NULL
  );

INSERT INTO schema_migrations(version)
VALUES ('202605150003_rbac_admin_management')
ON CONFLICT DO NOTHING;
