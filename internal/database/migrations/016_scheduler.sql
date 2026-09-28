CREATE TABLE scheduler_state (
 id integer PRIMARY KEY CHECK (id = 1),
 document jsonb NOT NULL
);
