-- Transcripts were appended under generated "### HH:MM" headings and thoughts were
-- titled with timestamps. Drop the headings and name each thought after its opening
-- words, leaving any title the user wrote alone.
UPDATE running_notes SET
 body=btrim(regexp_replace(regexp_replace(body, '^###[ \t]*[0-9]{1,2}:[0-9]{2}[ \t]*$', '', 'gn'), '\n{3,}', E'\n\n', 'g'), E' \t\n'),
 revision=gen_random_uuid()::text
WHERE body ~ '(^|\n)###[ \t]*[0-9]{1,2}:[0-9]{2}[ \t]*(\n|$)';

WITH words AS (
 SELECT id, regexp_split_to_array(btrim(body), '\s+') AS w FROM running_notes
 WHERE title ~ '^(Run [0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}|Audio thought [0-9]{4}-[0-9]{2}-[0-9]{2})$'
), titled AS (
 SELECT id, CASE WHEN w = '{""}' THEN 'New audio thought'
  WHEN cardinality(w) > 6 OR length(array_to_string(w[1:6], ' ')) > 60
   THEN rtrim(left(array_to_string(w[1:6], ' '), 60), ' .,;:!?-') || '...'
  ELSE array_to_string(w, ' ') END AS title
 FROM words
)
UPDATE running_notes n SET title=t.title, revision=gen_random_uuid()::text FROM titled t WHERE n.id=t.id;
