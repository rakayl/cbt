CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

DO $$ BEGIN
  CREATE TYPE subscription_tier AS ENUM ('free','basic','professional','enterprise');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
  CREATE TYPE exam_session_status AS ENUM ('created','started','reconnecting','submitted','completed','expired','cancelled');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now());

CREATE TABLE IF NOT EXISTS auth (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'auth',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_auth_tenant_deleted ON auth(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_auth_status ON auth(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_auth_search ON auth USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_auth_metadata ON auth USING gin(metadata);

CREATE TABLE IF NOT EXISTS tenants (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'tenants',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_tenants_tenant_deleted ON tenants(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_search ON tenants USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_tenants_metadata ON tenants USING gin(metadata);

CREATE TABLE IF NOT EXISTS subscription_plans (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'subscription_plans',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_subscription_plans_tenant_deleted ON subscription_plans(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_subscription_plans_status ON subscription_plans(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_subscription_plans_search ON subscription_plans USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_subscription_plans_metadata ON subscription_plans USING gin(metadata);

CREATE TABLE IF NOT EXISTS users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'users',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_users_tenant_deleted ON users(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_search ON users USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_users_metadata ON users USING gin(metadata);

CREATE TABLE IF NOT EXISTS roles (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'roles',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_roles_tenant_deleted ON roles(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_roles_status ON roles(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_roles_search ON roles USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_roles_metadata ON roles USING gin(metadata);

CREATE TABLE IF NOT EXISTS permissions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'permissions',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_permissions_tenant_deleted ON permissions(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_permissions_status ON permissions(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_permissions_search ON permissions USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_permissions_metadata ON permissions USING gin(metadata);

CREATE TABLE IF NOT EXISTS role_permissions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'role_permissions',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_role_permissions_tenant_deleted ON role_permissions(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_role_permissions_status ON role_permissions(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_role_permissions_search ON role_permissions USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_role_permissions_metadata ON role_permissions USING gin(metadata);

CREATE TABLE IF NOT EXISTS user_roles (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'user_roles',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_user_roles_tenant_deleted ON user_roles(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_user_roles_status ON user_roles(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_roles_search ON user_roles USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_user_roles_metadata ON user_roles USING gin(metadata);

CREATE TABLE IF NOT EXISTS user_sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'user_sessions',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_user_sessions_tenant_deleted ON user_sessions(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_user_sessions_status ON user_sessions(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_sessions_search ON user_sessions USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_user_sessions_metadata ON user_sessions USING gin(metadata);

CREATE TABLE IF NOT EXISTS login_histories (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'login_histories',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_login_histories_tenant_deleted ON login_histories(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_login_histories_status ON login_histories(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_login_histories_search ON login_histories USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_login_histories_metadata ON login_histories USING gin(metadata);

CREATE TABLE IF NOT EXISTS campuses (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'campuses',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_campuses_tenant_deleted ON campuses(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_campuses_status ON campuses(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_campuses_search ON campuses USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_campuses_metadata ON campuses USING gin(metadata);

CREATE TABLE IF NOT EXISTS faculties (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'faculties',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_faculties_tenant_deleted ON faculties(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_faculties_status ON faculties(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_faculties_search ON faculties USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_faculties_metadata ON faculties USING gin(metadata);

CREATE TABLE IF NOT EXISTS study_programs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'study_programs',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_study_programs_tenant_deleted ON study_programs(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_study_programs_status ON study_programs(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_study_programs_search ON study_programs USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_study_programs_metadata ON study_programs USING gin(metadata);

CREATE TABLE IF NOT EXISTS academic_periods (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'academic_periods',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_academic_periods_tenant_deleted ON academic_periods(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_academic_periods_status ON academic_periods(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_academic_periods_search ON academic_periods USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_academic_periods_metadata ON academic_periods USING gin(metadata);

CREATE TABLE IF NOT EXISTS students (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'students',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_students_tenant_deleted ON students(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_students_status ON students(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_students_search ON students USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_students_metadata ON students USING gin(metadata);

CREATE TABLE IF NOT EXISTS lecturers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'lecturers',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_lecturers_tenant_deleted ON lecturers(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_lecturers_status ON lecturers(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_lecturers_search ON lecturers USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_lecturers_metadata ON lecturers USING gin(metadata);

CREATE TABLE IF NOT EXISTS courses (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'courses',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_courses_tenant_deleted ON courses(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_courses_status ON courses(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_courses_search ON courses USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_courses_metadata ON courses USING gin(metadata);

CREATE TABLE IF NOT EXISTS class_rooms (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'class_rooms',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_class_rooms_tenant_deleted ON class_rooms(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_class_rooms_status ON class_rooms(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_class_rooms_search ON class_rooms USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_class_rooms_metadata ON class_rooms USING gin(metadata);

CREATE TABLE IF NOT EXISTS course_classes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'course_classes',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_course_classes_tenant_deleted ON course_classes(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_course_classes_status ON course_classes(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_course_classes_search ON course_classes USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_course_classes_metadata ON course_classes USING gin(metadata);

CREATE TABLE IF NOT EXISTS course_students (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'course_students',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_course_students_tenant_deleted ON course_students(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_course_students_status ON course_students(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_course_students_search ON course_students USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_course_students_metadata ON course_students USING gin(metadata);

CREATE TABLE IF NOT EXISTS enrollment (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'enrollment',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_enrollment_tenant_deleted ON enrollment(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_enrollment_status ON enrollment(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_enrollment_search ON enrollment USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_enrollment_metadata ON enrollment USING gin(metadata);

CREATE TABLE IF NOT EXISTS question_categories (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'question_categories',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_question_categories_tenant_deleted ON question_categories(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_question_categories_status ON question_categories(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_question_categories_search ON question_categories USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_question_categories_metadata ON question_categories USING gin(metadata);

CREATE TABLE IF NOT EXISTS question_banks (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'question_banks',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_question_banks_tenant_deleted ON question_banks(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_question_banks_status ON question_banks(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_question_banks_search ON question_banks USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_question_banks_metadata ON question_banks USING gin(metadata);

CREATE TABLE IF NOT EXISTS questions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'questions',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_questions_tenant_deleted ON questions(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_questions_status ON questions(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_questions_search ON questions USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_questions_metadata ON questions USING gin(metadata);

CREATE TABLE IF NOT EXISTS question_options (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'question_options',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_question_options_tenant_deleted ON question_options(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_question_options_status ON question_options(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_question_options_search ON question_options USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_question_options_metadata ON question_options USING gin(metadata);

CREATE TABLE IF NOT EXISTS question_tags (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'question_tags',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_question_tags_tenant_deleted ON question_tags(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_question_tags_status ON question_tags(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_question_tags_search ON question_tags USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_question_tags_metadata ON question_tags USING gin(metadata);

CREATE TABLE IF NOT EXISTS question_tag_relations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'question_tag_relations',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_question_tag_relations_tenant_deleted ON question_tag_relations(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_question_tag_relations_status ON question_tag_relations(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_question_tag_relations_search ON question_tag_relations USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_question_tag_relations_metadata ON question_tag_relations USING gin(metadata);

CREATE TABLE IF NOT EXISTS exams (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'exams',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_exams_tenant_deleted ON exams(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_exams_status ON exams(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_exams_search ON exams USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_exams_metadata ON exams USING gin(metadata);

CREATE TABLE IF NOT EXISTS exam_question_pools (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'exam_question_pools',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_exam_question_pools_tenant_deleted ON exam_question_pools(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_exam_question_pools_status ON exam_question_pools(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_exam_question_pools_search ON exam_question_pools USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_exam_question_pools_metadata ON exam_question_pools USING gin(metadata);

CREATE TABLE IF NOT EXISTS exam_sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'exam_sessions',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_exam_sessions_tenant_deleted ON exam_sessions(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_exam_sessions_status ON exam_sessions(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_exam_sessions_search ON exam_sessions USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_exam_sessions_metadata ON exam_sessions USING gin(metadata);

CREATE TABLE IF NOT EXISTS exam_session_questions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'exam_session_questions',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_exam_session_questions_tenant_deleted ON exam_session_questions(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_exam_session_questions_status ON exam_session_questions(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_exam_session_questions_search ON exam_session_questions USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_exam_session_questions_metadata ON exam_session_questions USING gin(metadata);

CREATE TABLE IF NOT EXISTS answers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'answers',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_answers_tenant_deleted ON answers(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_answers_status ON answers(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_answers_search ON answers USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_answers_metadata ON answers USING gin(metadata);

CREATE TABLE IF NOT EXISTS grading (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'grading',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_grading_tenant_deleted ON grading(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_grading_status ON grading(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_grading_search ON grading USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_grading_metadata ON grading USING gin(metadata);

CREATE TABLE IF NOT EXISTS analytics (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'analytics',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_analytics_tenant_deleted ON analytics(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_analytics_status ON analytics(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_analytics_search ON analytics USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_analytics_metadata ON analytics USING gin(metadata);

CREATE TABLE IF NOT EXISTS reports (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'reports',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_reports_tenant_deleted ON reports(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_reports_search ON reports USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_reports_metadata ON reports USING gin(metadata);

CREATE TABLE IF NOT EXISTS notifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'notifications',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_notifications_tenant_deleted ON notifications(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_search ON notifications USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_notifications_metadata ON notifications USING gin(metadata);

CREATE TABLE IF NOT EXISTS proctoring (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'proctoring',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_proctoring_tenant_deleted ON proctoring(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_proctoring_status ON proctoring(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_proctoring_search ON proctoring USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_proctoring_metadata ON proctoring USING gin(metadata);

CREATE TABLE IF NOT EXISTS audit_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'audit_logs',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_deleted ON audit_logs(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_status ON audit_logs(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_audit_logs_search ON audit_logs USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_audit_logs_metadata ON audit_logs USING gin(metadata);

CREATE TABLE IF NOT EXISTS recovery_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'recovery_logs',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_recovery_logs_tenant_deleted ON recovery_logs(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_recovery_logs_status ON recovery_logs(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_recovery_logs_search ON recovery_logs USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_recovery_logs_metadata ON recovery_logs USING gin(metadata);

CREATE TABLE IF NOT EXISTS reconnect_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'reconnect_logs',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_reconnect_logs_tenant_deleted ON reconnect_logs(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_reconnect_logs_status ON reconnect_logs(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_reconnect_logs_search ON reconnect_logs USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_reconnect_logs_metadata ON reconnect_logs USING gin(metadata);

CREATE TABLE IF NOT EXISTS proctoring_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'proctoring_logs',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_proctoring_logs_tenant_deleted ON proctoring_logs(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_proctoring_logs_status ON proctoring_logs(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_proctoring_logs_search ON proctoring_logs USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_proctoring_logs_metadata ON proctoring_logs USING gin(metadata);

CREATE TABLE IF NOT EXISTS browser_activity_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'browser_activity_logs',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_browser_activity_logs_tenant_deleted ON browser_activity_logs(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_browser_activity_logs_status ON browser_activity_logs(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_browser_activity_logs_search ON browser_activity_logs USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_browser_activity_logs_metadata ON browser_activity_logs USING gin(metadata);

CREATE TABLE IF NOT EXISTS face_detection_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'face_detection_logs',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_face_detection_logs_tenant_deleted ON face_detection_logs(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_face_detection_logs_status ON face_detection_logs(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_face_detection_logs_search ON face_detection_logs USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_face_detection_logs_metadata ON face_detection_logs USING gin(metadata);

CREATE TABLE IF NOT EXISTS screen_recordings (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'screen_recordings',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_screen_recordings_tenant_deleted ON screen_recordings(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_screen_recordings_status ON screen_recordings(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_screen_recordings_search ON screen_recordings USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_screen_recordings_metadata ON screen_recordings USING gin(metadata);

CREATE TABLE IF NOT EXISTS analytics_daily (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'analytics_daily',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_analytics_daily_tenant_deleted ON analytics_daily(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_analytics_daily_status ON analytics_daily(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_analytics_daily_search ON analytics_daily USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_analytics_daily_metadata ON analytics_daily USING gin(metadata);

CREATE TABLE IF NOT EXISTS billing (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NULL,
  code varchar(80) NOT NULL DEFAULT encode(gen_random_bytes(6), 'hex'),
  name varchar(160) NOT NULL DEFAULT 'billing',
  description text NULL,
  status varchar(40) NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_billing_tenant_deleted ON billing(tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_billing_status ON billing(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_billing_search ON billing USING gin (to_tsvector('simple', coalesce(code,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'')));
CREATE INDEX IF NOT EXISTS idx_billing_metadata ON billing USING gin(metadata);

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS domain varchar(255);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS custom_domain varchar(255);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS database_strategy varchar(32) NOT NULL DEFAULT 'shared';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS dedicated_database_url text;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS subscription_plan_id uuid;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS feature_flags jsonb NOT NULL DEFAULT '{}'::jsonb;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenants_domain ON tenants(domain) WHERE deleted_at IS NULL AND domain IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenants_custom_domain ON tenants(custom_domain) WHERE deleted_at IS NULL AND custom_domain IS NOT NULL;

ALTER TABLE subscription_plans ADD COLUMN IF NOT EXISTS tier subscription_tier NOT NULL DEFAULT 'free';
ALTER TABLE subscription_plans ADD COLUMN IF NOT EXISTS monthly_price_cents integer NOT NULL DEFAULT 0;
ALTER TABLE subscription_plans ADD COLUMN IF NOT EXISTS max_students integer NOT NULL DEFAULT 100;
ALTER TABLE subscription_plans ADD COLUMN IF NOT EXISTS max_concurrent_exams integer NOT NULL DEFAULT 10;
ALTER TABLE subscription_plans ADD COLUMN IF NOT EXISTS features jsonb NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE users ADD COLUMN IF NOT EXISTS email citext;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash text;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_enabled boolean NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS sso_provider varchar(80);
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at timestamptz;
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_tenant_email ON users(tenant_id, email) WHERE deleted_at IS NULL;
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS resource varchar(120);
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS action varchar(80);
ALTER TABLE role_permissions ADD COLUMN IF NOT EXISTS role_id uuid;
ALTER TABLE role_permissions ADD COLUMN IF NOT EXISTS permission_id uuid;
ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS user_id uuid;
ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS role_id uuid;
CREATE UNIQUE INDEX IF NOT EXISTS uq_role_permissions ON role_permissions(role_id, permission_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_roles ON user_roles(user_id, role_id) WHERE deleted_at IS NULL;
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS user_id uuid;
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS refresh_token_hash text;
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS device_name varchar(160);
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS fingerprint text;
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS ip_address inet;
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS user_agent text;
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS expires_at timestamptz;
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS revoked_at timestamptz;
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_active ON user_sessions(user_id, revoked_at, expires_at);
ALTER TABLE login_histories ADD COLUMN IF NOT EXISTS user_id uuid;
ALTER TABLE login_histories ADD COLUMN IF NOT EXISTS ip_address inet;
ALTER TABLE login_histories ADD COLUMN IF NOT EXISTS user_agent text;
ALTER TABLE login_histories ADD COLUMN IF NOT EXISTS success boolean NOT NULL DEFAULT false;
ALTER TABLE login_histories ADD COLUMN IF NOT EXISTS failure_reason text;
ALTER TABLE faculties ADD COLUMN IF NOT EXISTS campus_id uuid;
ALTER TABLE study_programs ADD COLUMN IF NOT EXISTS faculty_id uuid;
ALTER TABLE students ADD COLUMN IF NOT EXISTS user_id uuid;
ALTER TABLE students ADD COLUMN IF NOT EXISTS study_program_id uuid;
ALTER TABLE students ADD COLUMN IF NOT EXISTS student_number varchar(80);
ALTER TABLE lecturers ADD COLUMN IF NOT EXISTS user_id uuid;
ALTER TABLE courses ADD COLUMN IF NOT EXISTS study_program_id uuid;
ALTER TABLE course_classes ADD COLUMN IF NOT EXISTS course_id uuid;
ALTER TABLE course_classes ADD COLUMN IF NOT EXISTS lecturer_id uuid;
ALTER TABLE course_students ADD COLUMN IF NOT EXISTS course_class_id uuid;
ALTER TABLE course_students ADD COLUMN IF NOT EXISTS student_id uuid;
ALTER TABLE questions ADD COLUMN IF NOT EXISTS question_bank_id uuid;
ALTER TABLE questions ADD COLUMN IF NOT EXISTS question_type varchar(40) NOT NULL DEFAULT 'multiple_choice';
ALTER TABLE questions ADD COLUMN IF NOT EXISTS difficulty varchar(40) NOT NULL DEFAULT 'medium';
ALTER TABLE questions ADD COLUMN IF NOT EXISTS content text;
ALTER TABLE questions ADD COLUMN IF NOT EXISTS explanation text;
ALTER TABLE questions ADD COLUMN IF NOT EXISTS version integer NOT NULL DEFAULT 1;
ALTER TABLE questions ADD COLUMN IF NOT EXISTS media jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE question_options ADD COLUMN IF NOT EXISTS question_id uuid;
ALTER TABLE question_options ADD COLUMN IF NOT EXISTS is_correct boolean NOT NULL DEFAULT false;
ALTER TABLE question_tags ADD COLUMN IF NOT EXISTS color varchar(24);
ALTER TABLE question_tag_relations ADD COLUMN IF NOT EXISTS question_id uuid;
ALTER TABLE question_tag_relations ADD COLUMN IF NOT EXISTS question_tag_id uuid;
CREATE UNIQUE INDEX IF NOT EXISTS uq_question_tag_relations ON question_tag_relations(question_id, question_tag_id) WHERE deleted_at IS NULL;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS course_class_id uuid;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS duration_minutes integer NOT NULL DEFAULT 120;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS passing_grade numeric(5,2) NOT NULL DEFAULT 60;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS exam_token varchar(64);
ALTER TABLE exams ADD COLUMN IF NOT EXISTS random_question boolean NOT NULL DEFAULT true;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS random_option boolean NOT NULL DEFAULT true;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS max_attempt integer NOT NULL DEFAULT 1;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS instruction text;
ALTER TABLE exams ADD COLUMN IF NOT EXISTS published_at timestamptz;
ALTER TABLE exam_question_pools ADD COLUMN IF NOT EXISTS exam_id uuid;
ALTER TABLE exam_question_pools ADD COLUMN IF NOT EXISTS question_bank_id uuid;
ALTER TABLE exam_question_pools ADD COLUMN IF NOT EXISTS question_count integer NOT NULL DEFAULT 0;
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS exam_id uuid;
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS student_id uuid;
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS started_at timestamptz;
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS ends_at timestamptz;
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS submitted_at timestamptz;
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS server_time_anchor timestamptz;
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS status_enum exam_session_status NOT NULL DEFAULT 'created';
ALTER TABLE exam_sessions ADD COLUMN IF NOT EXISTS attempt integer NOT NULL DEFAULT 1;
ALTER TABLE exam_session_questions ADD COLUMN IF NOT EXISTS exam_session_id uuid;
ALTER TABLE exam_session_questions ADD COLUMN IF NOT EXISTS question_id uuid;
ALTER TABLE exam_session_questions ADD COLUMN IF NOT EXISTS position integer NOT NULL DEFAULT 0;
ALTER TABLE exam_session_questions ADD COLUMN IF NOT EXISTS option_order jsonb NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE answers ADD COLUMN IF NOT EXISTS exam_session_id uuid;
ALTER TABLE answers ADD COLUMN IF NOT EXISTS question_id uuid;
ALTER TABLE answers ADD COLUMN IF NOT EXISTS answer_payload jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE answers ADD COLUMN IF NOT EXISTS autosaved_at timestamptz;
ALTER TABLE answers ADD COLUMN IF NOT EXISTS submitted_at timestamptz;
CREATE UNIQUE INDEX IF NOT EXISTS uq_answers_session_question ON answers(exam_session_id, question_id) WHERE deleted_at IS NULL;
ALTER TABLE recovery_logs ADD COLUMN IF NOT EXISTS exam_session_id uuid;
ALTER TABLE recovery_logs ADD COLUMN IF NOT EXISTS event_type varchar(80);
ALTER TABLE recovery_logs ADD COLUMN IF NOT EXISTS client_time timestamptz;
ALTER TABLE reconnect_logs ADD COLUMN IF NOT EXISTS exam_session_id uuid;
ALTER TABLE reconnect_logs ADD COLUMN IF NOT EXISTS disconnected_at timestamptz;
ALTER TABLE reconnect_logs ADD COLUMN IF NOT EXISTS reconnected_at timestamptz;
ALTER TABLE reconnect_logs ADD COLUMN IF NOT EXISTS auto_submitted boolean NOT NULL DEFAULT false;
ALTER TABLE proctoring_logs ADD COLUMN IF NOT EXISTS exam_session_id uuid;
ALTER TABLE proctoring_logs ADD COLUMN IF NOT EXISTS event_type varchar(80);
ALTER TABLE proctoring_logs ADD COLUMN IF NOT EXISTS score numeric(6,2) NOT NULL DEFAULT 0;
ALTER TABLE browser_activity_logs ADD COLUMN IF NOT EXISTS exam_session_id uuid;
ALTER TABLE browser_activity_logs ADD COLUMN IF NOT EXISTS activity_type varchar(80);
ALTER TABLE face_detection_logs ADD COLUMN IF NOT EXISTS exam_session_id uuid;
ALTER TABLE face_detection_logs ADD COLUMN IF NOT EXISTS face_count integer NOT NULL DEFAULT 0;
ALTER TABLE screen_recordings ADD COLUMN IF NOT EXISTS exam_session_id uuid;
ALTER TABLE screen_recordings ADD COLUMN IF NOT EXISTS object_key text;
ALTER TABLE analytics_daily ADD COLUMN IF NOT EXISTS date date NOT NULL DEFAULT current_date;
ALTER TABLE analytics_daily ADD COLUMN IF NOT EXISTS metrics jsonb NOT NULL DEFAULT '{}'::jsonb;
CREATE UNIQUE INDEX IF NOT EXISTS uq_analytics_daily_tenant_date ON analytics_daily(tenant_id, date);
ALTER TABLE reports ADD COLUMN IF NOT EXISTS report_type varchar(80);
ALTER TABLE reports ADD COLUMN IF NOT EXISTS object_key text;
INSERT INTO subscription_plans(code,name,tier,monthly_price_cents,max_students,max_concurrent_exams,features) VALUES
('FREE','Free','free',0,100,10,'{"question_bank":true,"proctoring":false}'::jsonb),
('BASIC','Basic','basic',990000,1000,100,'{"question_bank":true,"proctoring":false,"reports":true}'::jsonb),
('PRO','Professional','professional',4990000,10000,1000,'{"question_bank":true,"proctoring":true,"analytics":true,"reports":true}'::jsonb),
('ENT','Enterprise','enterprise',0,100000,10000,'{"question_bank":true,"proctoring":true,"analytics":true,"reports":true,"custom_domain":true,"dedicated_database":true}'::jsonb)
ON CONFLICT DO NOTHING;
INSERT INTO schema_migrations(version) VALUES ('202605130001_enterprise_cbt_schema') ON CONFLICT DO NOTHING;
