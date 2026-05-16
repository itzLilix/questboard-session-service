ALTER TABLE characters
    DROP COLUMN sheet_url,
    DROP COLUMN description,
    DROP COLUMN avatar_url,
    ALTER COLUMN level SET DEFAULT 1;