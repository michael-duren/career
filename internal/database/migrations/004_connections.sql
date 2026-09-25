-- Connections are people, optionally linked to a tracked company. They replace
-- company_contacts so one person has one record across the companies and
-- connections pages.
CREATE TABLE connections (
 id UUID PRIMARY KEY,
 name TEXT NOT NULL CHECK(length(name) BETWEEN 1 AND 200),
 role TEXT NOT NULL CHECK(length(role)<=200),
 company_name TEXT NOT NULL CHECK(length(company_name)<=200),
 company_slug TEXT REFERENCES companies(slug) ON DELETE SET NULL ON UPDATE CASCADE,
 email TEXT NOT NULL CHECK(length(email)<=254),
 url TEXT,
 connected_on DATE,
 last_contacted_on DATE,
 cadence_days INTEGER CHECK(cadence_days BETWEEN 1 AND 3650),
 queued BOOLEAN NOT NULL,
 notes TEXT NOT NULL CHECK(length(notes)<=5000),
 tags TEXT[] NOT NULL CHECK(cardinality(tags)<=30),
 revision TEXT NOT NULL,
 position BIGINT NOT NULL CHECK(position>=0),
 updated_at TIMESTAMPTZ
);
-- LinkedIn imports upsert on the profile URL.
CREATE UNIQUE INDEX connections_url_idx ON connections(lower(url)) WHERE url IS NOT NULL;
CREATE INDEX connections_company_idx ON connections(company_slug, name);
CREATE INDEX connections_name_idx ON connections(name, id);
CREATE TABLE connection_photos (
 connection_id UUID PRIMARY KEY REFERENCES connections(id) ON DELETE CASCADE,
 mime_type TEXT NOT NULL CHECK(mime_type IN ('image/jpeg','image/png','image/webp','image/gif')),
 photo BYTEA NOT NULL CHECK(octet_length(photo) BETWEEN 1 AND 2097152),
 revision TEXT NOT NULL
);
-- Legacy contacts only had length checks. Values that would fail connection
-- validation move into notes so the row stays editable, a repeated profile URL
-- stays only on its first row, and a UUID reused across companies gets a new id.
INSERT INTO connections(id,name,role,company_name,company_slug,email,url,queued,notes,tags,revision,position,updated_at)
 SELECT CASE WHEN id_rank=1 THEN id ELSE gen_random_uuid() END, name, role, title, slug,
  CASE WHEN email_ok THEN email ELSE '' END,
  CASE WHEN url_ok AND url_rank=1 THEN url END, false,
  left(concat_ws(E'\n', NULLIF(notes,''),
   CASE WHEN url<>'' AND NOT url_ok THEN 'Profile: ' || url END,
   CASE WHEN email<>'' AND NOT email_ok THEN 'Email: ' || email END), 5000),
  '{}', gen_random_uuid()::text, nextval('entity_position'), updated_at
 FROM (
  SELECT cc.id, cc.name, cc.role, cc.email, cc.url, cc.notes, c.title, c.slug, c.updated_at, c.position AS company_position, cc.position AS contact_position,
   cc.url ~ '^https?://[^/\s]+[^\s]*$' AS url_ok,
   cc.email ~ '^[^@\s<>(),;:"]+@[^@\s<>(),;:"]+\.[^@\s<>(),;:"]+$' AS email_ok,
   row_number() OVER (PARTITION BY cc.id ORDER BY c.position, cc.position) AS id_rank,
   row_number() OVER (PARTITION BY lower(NULLIF(cc.url,'')) ORDER BY c.position, cc.position) AS url_rank
  FROM company_contacts cc JOIN companies c ON c.slug=cc.company_slug
 ) legacy
 ORDER BY company_position, contact_position;
-- Company entities lose their contacts field, so open editors must reload.
UPDATE companies SET revision=gen_random_uuid()::text WHERE contacts_present;
DROP TABLE company_contacts;
ALTER TABLE companies DROP COLUMN contacts_present;
