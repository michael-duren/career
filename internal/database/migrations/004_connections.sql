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
INSERT INTO connections(id,name,role,company_name,company_slug,email,url,queued,notes,tags,revision,position,updated_at)
 -- A repeated profile URL keeps its notes but only the first row keeps the URL.
 SELECT cc.id, cc.name, cc.role, c.title, c.slug, cc.email,
  CASE WHEN row_number() OVER (PARTITION BY lower(NULLIF(cc.url,'')) ORDER BY c.position, cc.position)=1 THEN NULLIF(cc.url,'') END, false, cc.notes, '{}', gen_random_uuid()::text, nextval('entity_position'), c.updated_at
 FROM company_contacts cc JOIN companies c ON c.slug=cc.company_slug
 ORDER BY c.position, cc.position
 ON CONFLICT DO NOTHING;
-- Company entities lose their contacts field, so open editors must reload.
UPDATE companies SET revision=gen_random_uuid()::text WHERE contacts_present;
DROP TABLE company_contacts;
ALTER TABLE companies DROP COLUMN contacts_present;
