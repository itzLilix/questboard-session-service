ALTER TABLE characters
    ALTER COLUMN level DROP DEFAULT,
    ADD COLUMN avatar_url  TEXT,
    ADD COLUMN description TEXT,
    ADD COLUMN sheet_url   TEXT;