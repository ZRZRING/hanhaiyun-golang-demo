-- Drop the triggers
DROP TRIGGER IF EXISTS set_updated_at_exam_items ON exam_items;
DROP TRIGGER IF EXISTS set_updated_at_student_answers ON student_answers;

-- Drop the function for updating the updated_at column
DROP FUNCTION IF EXISTS update_updated_at_column;

-- Drop the tables
DROP TABLE IF EXISTS student_answers;
DROP TABLE IF EXISTS exam_items;

-- Drop the pgcrypto extension