ALTER TABLE exams ADD COLUMN IF NOT EXISTS owner_user_id uuid;
CREATE INDEX IF NOT EXISTS idx_exams_owner ON exams(tenant_id, owner_user_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_exams_tenant_token ON exams(tenant_id, lower(exam_token)) WHERE deleted_at IS NULL AND exam_token IS NOT NULL;

CREATE TABLE IF NOT EXISTS exam_invites (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  exam_id uuid NOT NULL,
  student_id uuid NOT NULL,
  invited_by_user_id uuid,
  invitation_code varchar(64) NOT NULL,
  status varchar(40) NOT NULL DEFAULT 'invited',
  accepted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_exam_invites_exam ON exam_invites(tenant_id, exam_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_exam_invites_student ON exam_invites(tenant_id, student_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_exam_invites_exam_student ON exam_invites(tenant_id, exam_id, student_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_exam_invites_code ON exam_invites(tenant_id, lower(invitation_code)) WHERE deleted_at IS NULL;

WITH perms(code,name,resource,action) AS (
  VALUES
    ('dashboard:read','Dashboard Read','dashboard','read'),
    ('students:read','Students Read','students','read'),
    ('students:write','Students Write','students','write'),
    ('lecturers:read','Lecturers Read','lecturers','read'),
    ('lecturers:write','Lecturers Write','lecturers','write'),
    ('study.programs:read','Study Programs Read','study.programs','read'),
    ('study.programs:write','Study Programs Write','study.programs','write'),
    ('class.rooms:read','Class Rooms Read','class.rooms','read'),
    ('class.rooms:write','Class Rooms Write','class.rooms','write'),
    ('enrollment:read','Enrollment Read','enrollment','read'),
    ('enrollment:write','Enrollment Write','enrollment','write'),
    ('courses:read','Courses Read','courses','read'),
    ('courses:write','Courses Write','courses','write'),
    ('question.banks:read','Question Banks Read','question.banks','read'),
    ('question.banks:write','Question Banks Write','question.banks','write'),
    ('questions:read','Questions Read','questions','read'),
    ('questions:write','Questions Write','questions','write'),
    ('exams:read','Exams Read','exams','read'),
    ('exams:write','Exams Write','exams','write'),
    ('exams:invite','Exams Invite','exams','invite'),
    ('exams:join','Exams Join','exams','join'),
    ('exam.sessions:read','Exam Sessions Read','exam.sessions','read'),
    ('exam.sessions:write','Exam Sessions Write','exam.sessions','write'),
    ('analytics:read','Analytics Read','analytics','read'),
    ('reports:read','Reports Read','reports','read'),
    ('proctoring:read','Proctoring Read','proctoring','read'),
    ('billing:read','Billing Read','billing','read')
)
INSERT INTO permissions(id,tenant_id,code,name,resource,action,status,metadata)
SELECT gen_random_uuid(), t.id, p.code, p.name, p.resource, p.action, 'active', '{"seed":true}'::jsonb
FROM tenants t
CROSS JOIN perms p
ON CONFLICT DO NOTHING;

WITH role_seed(code,name) AS (
  VALUES ('LECTURER','Lecturer'), ('STUDENT','Student')
)
INSERT INTO roles(id,tenant_id,code,name,status,metadata)
SELECT gen_random_uuid(), t.id, r.code, r.name, 'active', '{"system":true,"academic_role":true}'::jsonb
FROM tenants t
CROSS JOIN role_seed r
WHERE NOT EXISTS (
  SELECT 1 FROM roles existing
  WHERE existing.tenant_id=t.id AND existing.code=r.code AND existing.deleted_at IS NULL
);

INSERT INTO role_permissions(id,tenant_id,code,name,role_id,permission_id,status,metadata)
SELECT gen_random_uuid(), r.tenant_id, r.code || '_' || p.code, r.name || ' ' || p.name, r.id, p.id, 'active', '{"seed":true}'::jsonb
FROM roles r
JOIN permissions p ON p.tenant_id=r.tenant_id AND p.deleted_at IS NULL
WHERE r.code='LECTURER'
  AND p.code IN ('dashboard:read','students:read','class.rooms:read','enrollment:read','courses:read','question.banks:read','question.banks:write','questions:read','questions:write','exams:read','exams:write','exams:invite','exam.sessions:read','analytics:read','reports:read','proctoring:read')
ON CONFLICT (role_id, permission_id) WHERE deleted_at IS NULL DO NOTHING;

INSERT INTO role_permissions(id,tenant_id,code,name,role_id,permission_id,status,metadata)
SELECT gen_random_uuid(), r.tenant_id, r.code || '_' || p.code, r.name || ' ' || p.name, r.id, p.id, 'active', '{"seed":true}'::jsonb
FROM roles r
JOIN permissions p ON p.tenant_id=r.tenant_id AND p.deleted_at IS NULL
WHERE r.code='STUDENT'
  AND p.code IN ('dashboard:read','exams:join','exam.sessions:read','exam.sessions:write')
ON CONFLICT (role_id, permission_id) WHERE deleted_at IS NULL DO NOTHING;

INSERT INTO schema_migrations(version)
VALUES ('202605130006_rbac_exam_invites')
ON CONFLICT DO NOTHING;
