ALTER TABLE exam_question_pools ADD COLUMN IF NOT EXISTS question_tag_id uuid;

CREATE INDEX IF NOT EXISTS idx_exam_question_pools_tag
ON exam_question_pools(tenant_id, exam_id, question_tag_id)
WHERE deleted_at IS NULL;

WITH perms(code,name,resource,action) AS (
  VALUES
    ('question.tags:read','Question Tags Read','question.tags','read'),
    ('question.tags:write','Question Tags Write','question.tags','write')
)
INSERT INTO permissions(id,tenant_id,code,name,resource,action,status,metadata)
SELECT gen_random_uuid(), t.id, p.code, p.name, p.resource, p.action, 'active', '{"seed":true}'::jsonb
FROM tenants t
CROSS JOIN perms p
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions(id,tenant_id,code,name,role_id,permission_id,status,metadata)
SELECT gen_random_uuid(), r.tenant_id, r.code || '_' || p.code, r.name || ' ' || p.name, r.id, p.id, 'active', '{"seed":true}'::jsonb
FROM roles r
JOIN permissions p ON p.tenant_id=r.tenant_id AND p.deleted_at IS NULL
WHERE r.code IN ('LECTURER')
  AND p.code IN ('question.tags:read','question.tags:write')
ON CONFLICT (role_id, permission_id) WHERE deleted_at IS NULL DO NOTHING;

INSERT INTO schema_migrations(version)
VALUES ('202605140002_question_tag_exam_pools')
ON CONFLICT DO NOTHING;
