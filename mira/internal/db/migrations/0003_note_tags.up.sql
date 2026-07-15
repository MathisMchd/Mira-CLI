CREATE TABLE note_tags (
    note_id uuid NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    tag     text NOT NULL,
    PRIMARY KEY (note_id, tag)
);

CREATE INDEX note_tags_tag_idx ON note_tags (tag);
