ALTER TABLE exam_blocks
    ADD COLUMN submit_id TEXT NOT NULL default '0', -- Add submit_id column with default value '0'
    ADD COLUMN score TEXT,
    ADD COLUMN full_score TEXT;