CREATE TABLE game_systems (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    slug           VARCHAR(100) NOT NULL UNIQUE,
    canonical_name VARCHAR(100) NOT NULL,
    is_curated     BOOLEAN      NOT NULL DEFAULT FALSE
);
