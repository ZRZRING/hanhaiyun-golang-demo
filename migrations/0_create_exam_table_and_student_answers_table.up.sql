-- Enable the pgcrypto extension for gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table for exam items
CREATE TABLE exam_items (
                            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), -- UUID as primary key
                            exam_id TEXT NOT NULL,                         -- Exam ID
                            item_id TEXT NOT NULL UNIQUE,                  -- Question ID
                            body TEXT NOT NULL,                            -- Question body
                            correct_answer TEXT NOT NULL,                  -- Correct answer
                            created_at TIMESTAMP DEFAULT NOW(),            -- Creation timestamp
                            updated_at TIMESTAMP DEFAULT NOW()             -- Last updated timestamp
);

-- Table for student answers
CREATE TABLE student_answers (
                                 id UUID PRIMARY KEY DEFAULT uuid_generate_v4(), -- UUID as primary key
                                 exam_id TEXT NOT NULL,                         -- Exam ID
                                 item_id TEXT NOT NULL,                         -- Question ID
                                 student_id TEXT NOT NULL,                      -- Student ID
                                 answer_list JSON NOT NULL,                    -- Student answers (JSON format)
                                 created_at TIMESTAMP DEFAULT NOW(),            -- Creation timestamp
                                 updated_at TIMESTAMP DEFAULT NOW()            -- Last updated timestamp
);

-- Create a function to automatically update the updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add triggers to update the updated_at column on row updates
CREATE TRIGGER set_updated_at_exam_items
    BEFORE UPDATE ON exam_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER set_updated_at_student_answers
    BEFORE UPDATE ON student_answers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();