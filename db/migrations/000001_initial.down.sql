-- Down migration for async-aware servers/requests/deliveries schema

BEGIN;

-- 1) Drop view first (depends on deliveries/requests)
DROP VIEW IF EXISTS request_rollup;

-- 2) Drop functions (must match exact signatures)

-- expand_request_to_deliveries(p_request_id bigint, p_dest_server_id integer, p_cc_servers integer[])
DROP FUNCTION IF EXISTS expand_request_to_deliveries(bigint, integer, integer[]);

-- claimers
DROP FUNCTION IF EXISTS claim_deliveries(integer);
DROP FUNCTION IF EXISTS claim_async_polls(integer);

-- async helpers
DROP FUNCTION IF EXISTS start_async_delivery(bigint, text, text, text, integer);
DROP FUNCTION IF EXISTS update_async_status(bigint, text, jsonb, text[], text[], integer);

-- finalizer
DROP FUNCTION IF EXISTS finalize_delivery(
    bigint,
    boolean,
    integer,
    text,
    text,
    timestamp with time zone,
    integer,
    integer,
    integer
);

-- 3) Drop tables (indexes and sequences go with them)

DROP TABLE IF EXISTS deliveries CASCADE;
DROP TABLE IF EXISTS requests CASCADE;
DROP TABLE IF EXISTS servers CASCADE;


DROP TABLE IF EXISTS user_permissions CASCADE;
DROP TABLE IF EXISTS user_role_permissions CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS user_roles CASCADE;
DROP TABLE IF EXISTS users CASCADE;

DROP EXTENSION IF EXISTS "uuid-ossp";

COMMIT;
