-- migrations/001_init.sql

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users (operator sekolah, admin, kepala sekolah)
CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255)        NOT NULL,
    email       VARCHAR(255) UNIQUE NOT NULL,
    password    TEXT                NOT NULL,
    role        VARCHAR(50)         NOT NULL DEFAULT 'operator', -- operator | admin | headmaster
    is_active   BOOLEAN             NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

-- Siswa
CREATE TABLE IF NOT EXISTS students (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nisn            VARCHAR(10) UNIQUE NOT NULL,
    full_name       VARCHAR(255)       NOT NULL,
    birth_place     VARCHAR(100),
    birth_date      DATE,
    gender          VARCHAR(10),        -- L | P
    class           VARCHAR(20),
    year_entry      INT,
    year_graduate   INT,
    photo_url       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Kategori prestasi
CREATE TABLE IF NOT EXISTS achievement_categories (
    id      SERIAL PRIMARY KEY,
    name    VARCHAR(100) NOT NULL,   -- e.g. "Akademik", "Non Akademik"
    type    VARCHAR(50)  NOT NULL    -- "academic" | "non_academic"
);

-- Level / tingkat lomba
CREATE TABLE IF NOT EXISTS competition_levels (
    id      SERIAL PRIMARY KEY,
    name    VARCHAR(100) NOT NULL,
    order_rank INT NOT NULL DEFAULT 0  -- untuk sorting (sekolah=1 ... internasional=6)
);

-- Prestasi siswa
CREATE TABLE IF NOT EXISTS achievements (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id       UUID         NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    competition_name VARCHAR(255) NOT NULL,
    organizer        VARCHAR(255) NOT NULL,
    category_id      INT          REFERENCES achievement_categories(id),
    rank             VARCHAR(50),                  -- Juara 1, 2, 3, Harapan I, dll
    level_id         INT          REFERENCES competition_levels(id),
    year             INT          NOT NULL,
    description      TEXT,
    created_by       UUID         REFERENCES users(id),
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Attachment / bukti prestasi
CREATE TABLE IF NOT EXISTS achievement_attachments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    achievement_id  UUID         NOT NULL REFERENCES achievements(id) ON DELETE CASCADE,
    file_url        TEXT         NOT NULL,
    file_name       VARCHAR(255),
    file_type       VARCHAR(50),   -- "image/jpeg", "application/pdf"
    label           VARCHAR(100),  -- "Piagam", "Foto", "Sertifikat"
    uploaded_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Surat keterangan prestasi
CREATE TABLE IF NOT EXISTS certificates (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id         UUID         NOT NULL REFERENCES students(id),
    certificate_number VARCHAR(100) UNIQUE NOT NULL,
    issued_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    issued_by          UUID         REFERENCES users(id),
    valid_until        DATE,
    qr_token           VARCHAR(255) UNIQUE NOT NULL,
    pdf_url            TEXT,
    status             VARCHAR(20)  NOT NULL DEFAULT 'active', -- active | revoked
    notes              TEXT,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Relasi sertifikat <-> prestasi (many-to-many)
CREATE TABLE IF NOT EXISTS certificate_achievements (
    certificate_id  UUID NOT NULL REFERENCES certificates(id) ON DELETE CASCADE,
    achievement_id  UUID NOT NULL REFERENCES achievements(id) ON DELETE CASCADE,
    PRIMARY KEY (certificate_id, achievement_id)
);

-- Seed: kategori prestasi
INSERT INTO achievement_categories (name, type) VALUES
    ('Akademik',     'academic'),
    ('Non Akademik', 'non_academic')
ON CONFLICT DO NOTHING;

-- Seed: tingkat lomba
INSERT INTO competition_levels (name, order_rank) VALUES
    ('Sekolah',        1),
    ('Kecamatan',      2),
    ('Kabupaten/Kota', 3),
    ('Provinsi',       4),
    ('Nasional',       5),
    ('Internasional',  6)
ON CONFLICT DO NOTHING;

-- Seed: default admin user (password: Admin@123)
-- password hash akan digenerate via aplikasi, ini placeholder
-- jalankan via seeder terpisah atau migration seeder