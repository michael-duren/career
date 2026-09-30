ALTER TABLE books DROP CONSTRAINT books_type_check;
ALTER TABLE books ADD CONSTRAINT books_type_check CHECK (type IN ('book', 'course', 'paper'));
