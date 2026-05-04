-- Restore one-per-student constraint (keeps most recent row per student).
DELETE FROM testimonials
WHERE id NOT IN (
    SELECT DISTINCT ON (student_id) id
    FROM testimonials
    ORDER BY student_id, created_at DESC
);
ALTER TABLE testimonials ADD CONSTRAINT testimonials_student_id_key UNIQUE (student_id);
