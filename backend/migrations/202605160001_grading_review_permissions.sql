WITH perms(code,name,resource,action) AS (
  VALUES
    ('grading:read','Grading Read','grading','read'),
    ('grading:write','Grading Write','grading','write')
)
INSERT INTO permissions(id,tenant_id,code,name,resource,action,status,metadata)
SELECT gen_random_uuid(), t.id, p.code, p.name, p.resource, p.action, 'active', '{"seed":true}'::jsonb
FROM tenants t
CROSS JOIN perms p
ON CONFLICT DO NOTHING;

WITH target_roles(code) AS (
  VALUES ('SUPER_ADMIN'), ('TENANT_ADMIN'), ('FACULTY_ADMIN'), ('LECTURER'), ('REVIEWER')
)
INSERT INTO role_permissions(id,tenant_id,code,name,role_id,permission_id,status,metadata)
SELECT gen_random_uuid(), r.tenant_id, r.code || '_' || p.code, r.name || ' ' || p.name, r.id, p.id, 'active', '{"seed":true}'::jsonb
FROM roles r
JOIN target_roles tr ON tr.code = r.code
JOIN permissions p ON p.tenant_id = r.tenant_id AND p.code IN ('grading:read','grading:write')
ON CONFLICT DO NOTHING;

INSERT INTO schema_migrations(version)
VALUES ('202605160001_grading_review_permissions')
ON CONFLICT DO NOTHING;
