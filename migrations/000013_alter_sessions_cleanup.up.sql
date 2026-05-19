ALTER TABLE sessions RENAME COLUMN durationHours TO duration_hours;
ALTER TABLE sessions ALTER COLUMN scheduled_at DROP NOT NULL;
