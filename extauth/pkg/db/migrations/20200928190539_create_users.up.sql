CREATE TYPE type AS ENUM ('member', 'creator');

CREATE TABLE IF NOT EXISTS public.users
(
    id VARCHAR(100) PRIMARY KEY,
    username VARCHAR(100) UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    password VARCHAR(100),
    given_name VARCHAR(100),
    family_name VARCHAR(100),
    email_verified BOOLEAN DEFAULT FALSE,
    type type NOT NULL DEFAULT 'member',
    password_changed_at TIMESTAMPTZ
);

CREATE INDEX ON users (id DESC, username DESC);
CREATE UNIQUE INDEX idx_lower_unique_username ON users (LOWER(username));
CREATE UNIQUE INDEX idx_lower_unique_email ON users (LOWER(email));

-- TODO: remove
INSERT INTO public.users VALUES ('ciccio', 'pasticcio', 'psw', 'name', 'family');