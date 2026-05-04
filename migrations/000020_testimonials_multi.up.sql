-- Allow multiple testimonials per student (one per 90-day window, enforced in handler).
ALTER TABLE testimonials DROP CONSTRAINT IF EXISTS testimonials_student_id_key;
