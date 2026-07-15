CREATE TABLE notes (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title             text NOT NULL,
    content           text NOT NULL,
    summary           text NOT NULL DEFAULT '',
    score             double precision NOT NULL DEFAULT 0,
    enrichment_status text NOT NULL DEFAULT 'pending'
                      CHECK (enrichment_status IN ('pending', 'done', 'failed')),
    search_vector     tsvector GENERATED ALWAYS AS (
                          to_tsvector('french', coalesce(title, '') || ' ' || coalesce(content, ''))
                      ) STORED,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX notes_search_vector_idx ON notes USING GIN (search_vector);
CREATE INDEX notes_enrichment_status_idx ON notes (enrichment_status);
