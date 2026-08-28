-- Runs automatically on first container start (postgres image convention:
-- anything mounted into /docker-entrypoint-initdb.d/ executes once, at init).

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    total_cents INTEGER NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO users (email)
SELECT 'user' || g || '@example.com'
FROM generate_series(1, 2000) AS g;

-- Deliberately no index on orders.user_id, so `WHERE user_id = $1` has to
-- sequential scan -- gives the future query collector something to flag.
INSERT INTO orders (user_id, total_cents, status, created_at)
SELECT
    (random() * 1999 + 1)::int,
    (random() * 10000)::int,
    (ARRAY['pending', 'paid', 'shipped', 'refunded'])[(random() * 3 + 1)::int],
    now() - (random() * interval '90 days')
FROM generate_series(1, 50000);

ANALYZE users;
ANALYZE orders;

-- Two more databases on the same server, so dbwatch's multi-database
-- selector has something real to switch between.
CREATE DATABASE dbwatch_staging;
CREATE DATABASE dbwatch_analytics;

\connect dbwatch_staging

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO users (email)
SELECT 'staging-user' || g || '@example.com'
FROM generate_series(1, 200) AS g;
ANALYZE users;

\connect dbwatch_analytics

CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO events (name, occurred_at)
SELECT
    (ARRAY['page_view', 'signup', 'purchase', 'click'])[(random() * 3 + 1)::int],
    now() - (random() * interval '30 days')
FROM generate_series(1, 20000);
ANALYZE events;
