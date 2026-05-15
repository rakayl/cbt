ALTER TABLE class_rooms ADD COLUMN IF NOT EXISTS lecturer_id uuid;
ALTER TABLE class_rooms ADD COLUMN IF NOT EXISTS owner_user_id uuid;

CREATE INDEX IF NOT EXISTS idx_class_rooms_lecturer
ON class_rooms(tenant_id, lecturer_id)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_class_rooms_owner_user
ON class_rooms(tenant_id, owner_user_id)
WHERE deleted_at IS NULL;

WITH owners AS (
  SELECT DISTINCT ON (id)
    id,
    NULLIF(metadata->>'lecturer_id', '')::uuid AS lecturer_id,
    NULLIF(metadata->>'owner_user_id', '')::uuid AS owner_user_id
  FROM class_rooms
  WHERE deleted_at IS NULL
    AND (metadata ? 'lecturer_id' OR metadata ? 'owner_user_id')
  ORDER BY id, updated_at DESC NULLS LAST, created_at DESC NULLS LAST
)
UPDATE class_rooms cr
SET lecturer_id = COALESCE(cr.lecturer_id, owners.lecturer_id),
    owner_user_id = COALESCE(cr.owner_user_id, owners.owner_user_id),
    metadata = COALESCE(cr.metadata, '{}'::jsonb) || jsonb_strip_nulls(jsonb_build_object(
      'lecturer_id', COALESCE(cr.lecturer_id, owners.lecturer_id),
      'owner_user_id', COALESCE(cr.owner_user_id, owners.owner_user_id)
    ))
FROM owners
WHERE cr.id = owners.id
  AND cr.deleted_at IS NULL
  AND (cr.lecturer_id IS NULL OR cr.owner_user_id IS NULL);

INSERT INTO permissions(code,name,resource,action)
SELECT 'class.rooms:write','Class Rooms Write','class.rooms','write'
WHERE NOT EXISTS (
  SELECT 1 FROM permissions WHERE code='class.rooms:write'
);

INSERT INTO role_permissions(role_id,permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code='class.rooms:write'
WHERE r.code='LECTURER'
  AND NOT EXISTS (
    SELECT 1
    FROM role_permissions rp
    WHERE rp.role_id=r.id AND rp.permission_id=p.id
  );

INSERT INTO schema_migrations(version)
VALUES ('202605150002_class_room_teacher_ownership')
ON CONFLICT DO NOTHING;
