CREATE TABLE api_calls (
    id SERIAL PRIMARY KEY,
    ctime timestamptz DEFAULT now() NOT NULL,

    video_id text DEFAULT '' NOT NULL,
    called timestamptz,
    taken real,
    type text DEFAULT '' NOT NULL,
    error text,
    result jsonb NOT NULL
);