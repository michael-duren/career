CREATE TABLE note_todos (
 note_id TEXT REFERENCES notes(id) ON DELETE CASCADE, id UUID, position INTEGER NOT NULL CHECK(position>=0),
 title TEXT NOT NULL CHECK(length(title) BETWEEN 1 AND 500), done BOOLEAN NOT NULL,
 PRIMARY KEY(note_id,id), UNIQUE(note_id,position)
);
