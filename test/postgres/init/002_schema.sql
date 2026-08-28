-- Schema only -- no data. Row counts are seeded separately and
-- configurably by `dbwatch-test seed` (see internal/testkit/seed.go), per
-- the "don't blindly insert millions of rows" instruction.

\connect dbwatch_test_production

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price_cents INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    total_cents INTEGER NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_orders_user_id ON orders(user_id);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id),
    product_id INTEGER NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL,
    unit_price_cents INTEGER NOT NULL
);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);

-- Deliberately NOT indexed on order_id -- gives the query collector a
-- real, reproducible sequential-scan case for the index-testing scenario.
CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id),
    amount_cents INTEGER NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    name TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_events_occurred_at ON events(occurred_at);

CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Dedicated table for the lock-contention / deadlock scenarios (matches
-- the example in the spec: two transactions fighting over accounts.id=1).
CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    owner TEXT NOT NULL,
    balance_cents INTEGER NOT NULL
);
INSERT INTO accounts (owner, balance_cents) VALUES
    ('alice', 100000),
    ('bob', 100000);

-- Restricted role can read the "customer-facing" tables but not the
-- financial/audit ones -- the permission-error scenario runs
-- `SELECT * FROM payments` as this role and expects it to fail.
GRANT CONNECT ON DATABASE dbwatch_test_production TO dbwatch_test_restricted;
GRANT USAGE ON SCHEMA public TO dbwatch_test_restricted;
GRANT SELECT ON users, products, orders, order_items TO dbwatch_test_restricted;

\connect dbwatch_test_staging

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    total_cents INTEGER NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_orders_user_id ON orders(user_id);

\connect dbwatch_test_development

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

\connect dbwatch_test_analytics

CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_events_occurred_at ON events(occurred_at);

CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

\connect dbwatch_test_stress

-- Isolated from "production" on purpose: connection-pressure, growth, and
-- high-volume workload scenarios hammer this database so the numbers on
-- the other four stay realistic and readable.
CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    owner TEXT NOT NULL,
    balance_cents INTEGER NOT NULL
);
INSERT INTO accounts (owner, balance_cents) VALUES ('alice', 100000), ('bob', 100000);

CREATE TABLE growth_test (
    id SERIAL PRIMARY KEY,
    payload TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
