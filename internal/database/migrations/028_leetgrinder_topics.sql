-- LeetCode topic display names, keyed by slug. TopicLabel title-cases a slug,
-- which gets "Depth First Search" where LeetCode says "Depth-First Search";
-- a stored name wins and TopicLabel is the fallback. Names are upserted
-- wherever problem metadata is saved from LeetCode (the extension and the
-- background fetcher); MCP todo topics carry only slugs. The rows below
-- backfill the common tags whose name is not the title-cased slug, and never
-- replace a name already stored.
CREATE TABLE leetgrinder_topics (
    slug TEXT PRIMARY KEY CHECK (char_length(slug) BETWEEN 1 AND 100),
    name TEXT NOT NULL CHECK (name <> '' AND char_length(name) <= 100)
);

INSERT INTO leetgrinder_topics(slug, name) VALUES
    ('depth-first-search', 'Depth-First Search'),
    ('breadth-first-search', 'Breadth-First Search'),
    ('divide-and-conquer', 'Divide and Conquer'),
    ('heap-priority-queue', 'Heap (Priority Queue)'),
    ('doubly-linked-list', 'Doubly-Linked List'),
    ('probability-and-statistics', 'Probability and Statistics')
ON CONFLICT (slug) DO NOTHING;
