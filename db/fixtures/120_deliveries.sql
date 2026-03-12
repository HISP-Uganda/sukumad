-- Expand requests into deliveries (idempotent: only for requests that lack deliveries)
WITH missing AS (
    SELECT r.id AS request_id, r.destination AS server_id
    FROM requests r
             LEFT JOIN deliveries d ON d.request_id = r.id AND d.server_id = r.destination
    WHERE d.request_id IS NULL
)
INSERT INTO deliveries (request_id, server_id, status, attempts, next_run_at, priority, scheduled_at, updated_at)
SELECT request_id, server_id, 'ready', 0, now(), 100, now(), now()
FROM missing
ON CONFLICT DO NOTHING;

-- CC deliveries
WITH cc AS (
    SELECT r.id AS request_id, unnest(r.cc_servers)::int AS server_id
    FROM requests r
), missing_cc AS (
    SELECT c.request_id, c.server_id
    FROM cc c
             LEFT JOIN deliveries d ON d.request_id = c.request_id AND d.server_id = c.server_id
    WHERE d.request_id IS NULL
)
INSERT INTO deliveries (request_id, server_id, status, attempts, next_run_at, priority, scheduled_at, updated_at)
SELECT request_id, server_id, 'ready', 0, now(), 90, now(), now()
FROM missing_cc
ON CONFLICT DO NOTHING;
