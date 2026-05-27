-- =========================================
-- Cloud-Report initial schema
-- =========================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ----- USERS / AUTH -----
CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid         TEXT UNIQUE NOT NULL,
    username        TEXT UNIQUE NOT NULL,
    email           TEXT UNIQUE,
    password_hash   TEXT NOT NULL,
    is_admin        BOOLEAN NOT NULL DEFAULT FALSE,
    read_all        BOOLEAN NOT NULL DEFAULT FALSE,
    edit_all        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    -- We only store the SHA-256 hash of the key. Prefix is shown to user (first 8 chars) for identification.
    key_prefix      TEXT NOT NULL,
    key_hash        TEXT NOT NULL UNIQUE,
    scopes          TEXT[] NOT NULL DEFAULT ARRAY['render','read','write'],
    last_used_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);

CREATE TABLE IF NOT EXISTS users_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid     TEXT UNIQUE NOT NULL,
    name        TEXT NOT NULL,
    is_admin    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users_groups_members (
    group_id  UUID NOT NULL REFERENCES users_groups(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id)
);

-- ----- FOLDERS -----
CREATE TABLE IF NOT EXISTS folders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid         TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    parent_id       UUID REFERENCES folders(id) ON DELETE CASCADE,
    owner_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    read_perms      TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    edit_perms      TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_id);

-- ----- TEMPLATES -----
CREATE TABLE IF NOT EXISTS templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid         TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    folder_id       UUID REFERENCES folders(id) ON DELETE SET NULL,
    content         TEXT NOT NULL DEFAULT '',
    engine          TEXT NOT NULL DEFAULT 'handlebars',
    recipe          TEXT NOT NULL DEFAULT 'html',
    helpers         TEXT NOT NULL DEFAULT '',
    data_shortid    TEXT,
    scripts         JSONB NOT NULL DEFAULT '[]'::jsonb,
    chrome          JSONB,
    weasyprint      JSONB,
    docx            JSONB,
    xlsx            JSONB,
    pptx            JSONB,
    pdf_operations  JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_public       BOOLEAN NOT NULL DEFAULT FALSE,
    owner_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    read_perms      TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    edit_perms      TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2026-05 — add separate CSS field + page settings (page_size, page_orientation,
-- page_margin) so users can manage style sheets in their own editor tab.
ALTER TABLE templates ADD COLUMN IF NOT EXISTS css             TEXT NOT NULL DEFAULT '';
ALTER TABLE templates ADD COLUMN IF NOT EXISTS page_size       TEXT NOT NULL DEFAULT 'A4';
ALTER TABLE templates ADD COLUMN IF NOT EXISTS page_orientation TEXT NOT NULL DEFAULT 'portrait';
ALTER TABLE templates ADD COLUMN IF NOT EXISTS page_margin     TEXT NOT NULL DEFAULT '1cm';
CREATE INDEX IF NOT EXISTS idx_templates_folder ON templates(folder_id);
CREATE INDEX IF NOT EXISTS idx_templates_name ON templates(name);

-- ----- ASSETS -----
CREATE TABLE IF NOT EXISTS assets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid         TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    folder_id       UUID REFERENCES folders(id) ON DELETE SET NULL,
    blob_key        TEXT,            -- key in SeaweedFS S3 (null if inline content for small text)
    inline_content  TEXT,
    mime_type       TEXT,
    size_bytes      BIGINT NOT NULL DEFAULT 0,
    is_shared_helper        BOOLEAN NOT NULL DEFAULT FALSE,
    shared_helpers_scope    TEXT,
    link            TEXT,
    owner_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_assets_folder ON assets(folder_id);

-- ----- SCRIPTS -----
CREATE TABLE IF NOT EXISTS scripts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid         TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    folder_id       UUID REFERENCES folders(id) ON DELETE SET NULL,
    content         TEXT NOT NULL DEFAULT '',
    scope           TEXT NOT NULL DEFAULT 'template',  -- template | global | folder
    is_global       BOOLEAN NOT NULL DEFAULT FALSE,
    owner_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ----- DATA ITEMS -----
CREATE TABLE IF NOT EXISTS data_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid         TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    folder_id       UUID REFERENCES folders(id) ON DELETE SET NULL,
    data_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
    owner_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ----- COMPONENTS -----
CREATE TABLE IF NOT EXISTS components (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid         TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    folder_id       UUID REFERENCES folders(id) ON DELETE SET NULL,
    content         TEXT NOT NULL DEFAULT '',
    engine          TEXT NOT NULL DEFAULT 'handlebars',
    helpers         TEXT NOT NULL DEFAULT '',
    owner_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ----- REPORTS -----
CREATE TABLE IF NOT EXISTS reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid         TEXT UNIQUE NOT NULL,
    name            TEXT,
    template_shortid TEXT,
    state           TEXT NOT NULL DEFAULT 'success',
    error           TEXT,
    mime_type       TEXT,
    blob_key        TEXT,
    size_bytes      BIGINT NOT NULL DEFAULT 0,
    is_public       BOOLEAN NOT NULL DEFAULT FALSE,
    task_id         TEXT,
    owner_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_reports_template ON reports(template_shortid);
CREATE INDEX IF NOT EXISTS idx_reports_created ON reports(created_at DESC);

-- ----- SCHEDULES -----
CREATE TABLE IF NOT EXISTS schedules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid         TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    cron            TEXT NOT NULL,
    template_shortid TEXT NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    state           TEXT NOT NULL DEFAULT 'planned',
    next_run        TIMESTAMPTZ,
    owner_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS schedule_tasks (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid           TEXT UNIQUE NOT NULL,
    schedule_shortid  TEXT NOT NULL,
    state             TEXT NOT NULL DEFAULT 'running',
    error             TEXT,
    started_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finish_date       TIMESTAMPTZ,
    ping              TIMESTAMPTZ
);

-- ----- PROFILES (renders) -----
CREATE TABLE IF NOT EXISTS profiles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid         TEXT UNIQUE NOT NULL,
    template_shortid TEXT,
    state           TEXT NOT NULL DEFAULT 'running',
    mode            TEXT NOT NULL DEFAULT 'standard',
    error           TEXT,
    blob_key        TEXT,         -- events log in SeaweedFS
    timeout_ms      INT NOT NULL DEFAULT 60000,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at     TIMESTAMPTZ,
    owner_id        UUID REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_profiles_template ON profiles(template_shortid);
CREATE INDEX IF NOT EXISTS idx_profiles_started ON profiles(started_at DESC);

-- ----- SETTINGS -----
CREATE TABLE IF NOT EXISTS settings (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key       TEXT UNIQUE NOT NULL,
    value     JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ----- VERSION CONTROL -----
CREATE TABLE IF NOT EXISTS versions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid     TEXT UNIQUE NOT NULL,
    message     TEXT NOT NULL,
    blob_key    TEXT NOT NULL,         -- ZIP snapshot
    author_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ----- TAGS -----
CREATE TABLE IF NOT EXISTS tags (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortid     TEXT UNIQUE NOT NULL,
    name        TEXT NOT NULL,
    color       TEXT NOT NULL DEFAULT '#888',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ===========================================================================
-- 2026-05 — Report retention. Each template defines how many days its
-- generated reports are kept; a background sweeper deletes expired reports
-- (DB row + SeaweedFS blob). Default 30 days.
-- ===========================================================================
ALTER TABLE templates ADD COLUMN IF NOT EXISTS report_retention_days INT NOT NULL DEFAULT 30;
ALTER TABLE reports   ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_reports_expires ON reports(expires_at);
