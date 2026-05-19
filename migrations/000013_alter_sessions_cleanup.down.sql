-- Reverting SET NOT NULL only succeeds if no draft rows have NULL scheduled_at.
-- Resolve those manually before downgrading.
ALTER TABLE sessions ALTER COLUMN scheduled_at SET NOT NULL;
ALTER TABLE sessions RENAME COLUMN duration_hours TO "durationHours";
