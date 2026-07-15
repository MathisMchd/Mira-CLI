CREATE TABLE note_embeddings (
    note_id    uuid PRIMARY KEY REFERENCES notes (id) ON DELETE CASCADE,
    embedding  vector(768) NOT NULL,
    model      text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX note_embeddings_embedding_idx ON note_embeddings
    USING hnsw (embedding vector_cosine_ops);
