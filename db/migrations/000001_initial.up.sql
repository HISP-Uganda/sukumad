CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS user_roles
(
    id          serial PRIMARY KEY,
    name        text        NOT NULL UNIQUE,
    description text,
    created     timestamptz NOT NULL DEFAULT now(),
    updated     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS permissions
(
    id            serial PRIMARY KEY,
    name          text        NOT NULL,
    code          text        NOT NULL UNIQUE, -- e.g., can_view_reporters
    system_module text        NOT NULL,        -- e.g., Reporters
    created       timestamptz NOT NULL DEFAULT now(),
    updated       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users
(
    id                 serial PRIMARY KEY,
    user_role          integer     REFERENCES user_roles (id) ON DELETE SET NULL,
    firstname          text        NOT NULL,
    lastname           text        NOT NULL,
    username           text        NOT NULL UNIQUE,
    telephone          text,
    password_hash      text        NOT NULL,
    email              text UNIQUE,
    allowed_ips        inet[],
    denied_ips         inet[],
    failed_attempts    integer     NOT NULL DEFAULT 0,
    transaction_limit  numeric(18, 2)       DEFAULT NULL,
    is_active          boolean     NOT NULL DEFAULT true,
    is_system_user     boolean     NOT NULL DEFAULT false,
    last_login         timestamptz,
    last_failed_at     timestamptz,
    locked_until       timestamptz,
    last_passwd_update timestamptz,
    created            timestamptz NOT NULL DEFAULT now(),
    updated            timestamptz NOT NULL DEFAULT now()
);

-- Direct user-permission grants
CREATE TABLE IF NOT EXISTS user_permissions
(
    user_id       integer NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    permission_id integer NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, permission_id)
);

-- Role-permission grants
CREATE TABLE IF NOT EXISTS user_role_permissions
(
    role_id       integer NOT NULL REFERENCES user_roles (id) ON DELETE CASCADE,
    permission_id integer NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Helpful indexes
CREATE INDEX IF NOT EXISTS ix_users_role ON users(user_role);
CREATE INDEX IF NOT EXISTS ix_users_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS ix_permissions_module ON permissions(system_module);

CREATE TABLE IF NOT EXISTS refresh_tokens
(
    id         bigserial PRIMARY KEY,
    user_id    integer     NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    jti        text        NOT NULL UNIQUE, -- token id
    issued_at  timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    revoked    boolean     NOT NULL DEFAULT false,
    user_agent text,
    ip_address text
);

CREATE INDEX IF NOT EXISTS ix_refresh_tokens_user ON refresh_tokens (user_id);

CREATE TABLE IF NOT EXISTS password_reset_tokens
(
    id         bigserial PRIMARY KEY,
    user_id    integer     NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash text        NOT NULL UNIQUE,
    issued_at  timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    ip_address text,
    user_agent text
);

-- ==============================
-- 1) Downstream systems (servers)
-- ==============================
DROP TABLE IF EXISTS servers CASCADE;
CREATE TABLE IF NOT EXISTS servers (
    id                      serial PRIMARY KEY,
    uid                     text UNIQUE DEFAULT uuid_generate_v4()::text,
    name                    text NOT NULL,
    username                text,
    password                text,
    auth_token              text,
    ipaddress               text,
    url                     text NOT NULL,
    callback_url            text,
    cc_urls                 text[],
    http_method             text NOT NULL DEFAULT 'POST',
    auth_method             text CHECK (auth_method IN ('none','basic','bearer')) DEFAULT 'none',
    system_type             text,
    endpoint_type           text,
    url_params              jsonb DEFAULT '{}'::jsonb,
    allow_callbacks         boolean DEFAULT false,
    allow_copies            boolean DEFAULT false,
    use_async               boolean DEFAULT false,
    use_ssl                 boolean DEFAULT true,
    parse_responses         boolean DEFAULT true,
    ssl_client_certkey_file text,
    start_submission_period integer,
    end_submission_period   integer,
    xml_response_xpath      text,
    json_response_xpath     text,
    suspended               boolean DEFAULT false,

    rps                     numeric,
    burst                   integer,
    max_concurrency         integer,
    timeout_ms              integer,
    headers                 jsonb DEFAULT '{}'::jsonb,  -- canonical headers
    default_content_type    text NOT NULL DEFAULT 'application/json',
    rate_class              text,

    created                 timestamptz NOT NULL DEFAULT now(),
    updated                 timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON servers (suspended);
CREATE INDEX ON servers (rate_class);

-- ==============================
-- 2) Logical request
-- ==============================
DROP TABLE IF EXISTS requests CASCADE;
CREATE TABLE requests (
    id                   bigserial PRIMARY KEY,
    uid                  text UNIQUE DEFAULT uuid_generate_v4()::text,
    source               integer,
    destination          integer,        -- default server_id
    cc_servers           integer[],
    depends_on           bigint,
    batchid              text,

    url_suffix           text,
    body                 text NOT NULL,
    body_is_query_param  boolean DEFAULT false,

    frequency_type       text,
    period               text,
    week                 text,
    month                text,
    year                 integer,
    msisdn               text,
    raw_msg              text,
    facility             text,
    district             text,
    report_type          text,
    object_type          text,
    extras               jsonb DEFAULT '{}'::jsonb,

    suspended            boolean DEFAULT false,
    idempotency_key      text UNIQUE,

    created              timestamptz NOT NULL DEFAULT now(),
    updated              timestamptz NOT NULL DEFAULT now()
);

-- ==============================
-- 3) Per-destination delivery (work queue)
-- Async fields INCLUDED here
-- ==============================
DROP TABLE IF EXISTS deliveries CASCADE;
CREATE TABLE deliveries (
    id                     bigserial PRIMARY KEY,
    request_id             bigint  NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    server_id              integer NOT NULL REFERENCES servers(id),

    status                 text NOT NULL DEFAULT 'ready'
        CHECK (status IN ('ready','processing','awaiting_async','done','failed')),
    attempts               integer NOT NULL DEFAULT 0,
    next_run_at            timestamptz NOT NULL DEFAULT now(),
    retry_after_at         timestamptz,
    last_error             text,
    status_code            integer,
    response_body          text,
    priority               integer NOT NULL DEFAULT 0,
    scheduled_at           timestamptz NOT NULL DEFAULT now(),
    sent_at                timestamptz,
    updated_at             timestamptz NOT NULL DEFAULT now(),

    rps_override           numeric,
    burst_override         integer,
    timeout_ms             integer,
    max_attempts           integer,
    rate_class             text,

    depends_on_delivery_id bigint REFERENCES deliveries(id),

    async_jobid            text,
    async_status           text,
    async_response         jsonb,
    async_poll_url         text,
    async_last_polled      timestamptz,

    UNIQUE (request_id, server_id)
);

CREATE INDEX ix_deliveries_ready_partial
    ON deliveries (next_run_at, id) WHERE status = 'ready';

CREATE INDEX ix_deliveries_awaiting_async
    ON deliveries (next_run_at, id) WHERE status = 'awaiting_async';

CREATE INDEX ix_deliveries_server_ready
    ON deliveries (server_id, next_run_at, id) WHERE status = 'ready';

CREATE INDEX ix_deliveries_priority
    ON deliveries (priority DESC, next_run_at, id);

-- ==============================
-- 4) Expander (request -> deliveries)
-- ==============================
CREATE OR REPLACE FUNCTION expand_request_to_deliveries(
    p_request_id bigint,
    p_dest_server_id integer,
    p_cc_servers integer[] DEFAULT NULL,
    p_priority integer DEFAULT 100,
    p_cc_priority integer DEFAULT 50
) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO deliveries (request_id, server_id, priority)
    VALUES (p_request_id, p_dest_server_id, p_priority);

    IF p_cc_servers IS NOT NULL AND array_length(p_cc_servers,1) IS NOT NULL THEN
        INSERT INTO deliveries (request_id, server_id, priority)
        SELECT p_request_id, sid, p_cc_priority
        FROM unnest(p_cc_servers) AS sid;
    END IF;
END $$;

-- ==============================
-- 5) Claimer for initial sends (non-async / initial POST)
-- ==============================
CREATE OR REPLACE FUNCTION claim_deliveries(p_limit integer)
    RETURNS TABLE (
                      delivery_id        bigint,
                      request_id         bigint,
                      server_id          integer,
                      method             text,
                      final_url          text,
                      body               text,
                      content_type       text,
                      headers            jsonb,
                      auth_method        text,
                      username           text,
                      password           text,
                      auth_token         text,
                      timeout_ms         integer,
                      rate_key           text,
                      rps                numeric,
                      burst              integer,
                      max_concurrency    integer,
                      server_use_async   boolean
                  ) LANGUAGE plpgsql AS $$
BEGIN
    RETURN QUERY
        WITH picked AS (
            SELECT d.id, d.server_id, d.request_id
            FROM deliveries d
                     JOIN servers s ON s.id = d.server_id
                     JOIN requests r ON r.id = d.request_id
            WHERE d.status = 'ready'
              AND d.next_run_at <= now()
              AND COALESCE(s.suspended,false) = false
              AND COALESCE(r.suspended,false) = false
            ORDER BY d.priority DESC, d.next_run_at, d.id
                FOR UPDATE SKIP LOCKED
            LIMIT p_limit
            )
            UPDATE deliveries d
                SET status='processing', updated_at=now()
                FROM picked p
                    JOIN servers  s ON s.id = p.server_id
                    JOIN requests r ON r.id = p.request_id
                WHERE d.id = p.id
                RETURNING
                    d.id AS delivery_id,
                    r.id AS request_id,
                    s.id AS server_id,
                    s.http_method AS method,
                    CASE WHEN r.url_suffix IS NULL OR r.url_suffix = '' THEN s.url ELSE s.url || r.url_suffix END AS final_url,
                    r.body,
                    s.default_content_type AS content_type,
                    COALESCE(s.headers, '{}'::jsonb) AS headers,
                    s.auth_method, s.username, s.password, s.auth_token,
                    COALESCE(d.timeout_ms, s.timeout_ms) AS timeout_ms,
                    COALESCE(d.rate_class, s.rate_class, 'server:'||s.id::text) AS rate_key,
                    COALESCE(d.rps_override, s.rps) AS rps,
                    COALESCE(d.burst_override, s.burst) AS burst,
                    s.max_concurrency,
                    s.use_async AS server_use_async;
END $$;

--
-- -- ==============================
-- -- 6) Claimer for async polls (jobs in awaiting_async)
-- -- Call this on a separate poller loop
-- -- ==============================
CREATE OR REPLACE FUNCTION claim_async_polls(p_limit integer)
    RETURNS TABLE (
                      delivery_id       bigint,
                      server_id         integer,
                      poll_url          text,
                      auth_method       text,
                      username          text,
                      password          text,
                      auth_token        text,
                      timeout_ms        integer,
                      rate_key          text,
                      rps               numeric,
                      burst             integer,
                      max_concurrency   integer
                  ) LANGUAGE plpgsql AS $$
BEGIN
    RETURN QUERY
        WITH picked AS (
            SELECT d.id, d.server_id
            FROM deliveries d
                     JOIN servers s ON s.id = d.server_id
            WHERE d.status = 'awaiting_async'
              AND d.next_run_at <= now()
              AND COALESCE(s.suspended,false) = false
            ORDER BY d.next_run_at, d.id
                FOR UPDATE SKIP LOCKED
            LIMIT p_limit
            )
            UPDATE deliveries d
                SET updated_at = now()
                FROM picked p
                    JOIN servers s ON s.id = p.server_id
                WHERE d.id = p.id
                RETURNING
                    d.id AS delivery_id,
                    d.server_id,
                    COALESCE(d.async_poll_url, s.url) AS poll_url,
                    s.auth_method, s.username, s.password, s.auth_token,
                    COALESCE(d.timeout_ms, s.timeout_ms) AS timeout_ms,
                    COALESCE(d.rate_class, s.rate_class, 'server:'||s.id::text) AS rate_key,
                    COALESCE(d.rps_override, s.rps) AS rps,
                    COALESCE(d.burst_override, s.burst) AS burst,
                    s.max_concurrency;
END $$;


-- ==============================
-- 7) Mark delivery as async-pending after initial submit
-- (store job id, poll url, next poll time, etc.)
-- ==============================
CREATE OR REPLACE FUNCTION start_async_delivery(
    p_delivery_id bigint,
    p_jobid text,
    p_poll_url text,
    p_initial_async_status text DEFAULT 'queued',
    p_first_poll_delay_seconds integer DEFAULT 5
) RETURNS void
    LANGUAGE plpgsql AS
$$
BEGIN
    UPDATE deliveries
    SET async_jobid    = p_jobid,
        async_poll_url = NULLIF(p_poll_url, ''),
        async_status   = p_initial_async_status,
        status         = 'awaiting_async',
        next_run_at    = now() + make_interval(secs => p_first_poll_delay_seconds),
        sent_at        = COALESCE(sent_at, now()),
        updated_at     = now()
    WHERE id = p_delivery_id;
END
$$;

-- ==============================
-- 8) Update async status after polling
-- If terminal -> done/failed; else reschedule another poll
-- ==============================
CREATE OR REPLACE FUNCTION update_async_status(
    p_delivery_id bigint,
    p_async_status text, -- e.g., 'COMPLETED','SUCCESS','FAILED','RUNNING'
    p_response jsonb,
    p_terminal_success_values text[] DEFAULT ARRAY ['COMPLETED','SUCCESS','OK'],
    p_terminal_fail_values text[] DEFAULT ARRAY ['FAILED','ERROR'],
    p_poll_again_seconds integer DEFAULT 5
) RETURNS void
    LANGUAGE plpgsql AS
$$
DECLARE
    v_is_success boolean := p_async_status = ANY (p_terminal_success_values);
    v_is_failure boolean := p_async_status = ANY (p_terminal_fail_values);
BEGIN
    IF v_is_success THEN
        UPDATE deliveries
        SET async_status   = p_async_status,
            async_response = p_response,
            status         = 'done',
            status_code    = 200,
            last_error     = NULL,
            sent_at        = COALESCE(sent_at, now()),
            updated_at     = now()
        WHERE id = p_delivery_id;
    ELSIF v_is_failure THEN
        UPDATE deliveries
        SET async_status   = p_async_status,
            async_response = p_response,
            status         = 'failed',
            status_code    = 500,
            last_error     = COALESCE(p_response ->> 'message', 'async failed'),
            updated_at     = now()
        WHERE id = p_delivery_id;
    ELSE
        UPDATE deliveries
        SET async_status     = p_async_status,
            async_response   = p_response,
            async_last_polled= now(),
            next_run_at      = now() + make_interval(secs => p_poll_again_seconds),
            updated_at       = now()
        WHERE id = p_delivery_id;
    END IF;
END
$$;

-- ==============================
-- 9) Finalize (non-async or terminal result)
-- keeps same signature as before
-- ==============================
CREATE OR REPLACE FUNCTION finalize_delivery(
    p_delivery_id bigint,
    p_ok boolean,
    p_status_code integer,
    p_response_body text,
    p_error text,
    p_retry_after_ts timestamptz,
    p_default_max_attempts integer DEFAULT 8,
    p_backoff_min_seconds integer DEFAULT 2,
    p_backoff_max_seconds integer DEFAULT 120
) RETURNS void
    LANGUAGE plpgsql AS
$$
DECLARE
    v_attempts     int;
    v_max_attempts int;
    v_next         timestamptz;
BEGIN
    SELECT attempts, COALESCE(max_attempts, p_default_max_attempts)
    INTO v_attempts, v_max_attempts
    FROM deliveries
    WHERE id = p_delivery_id FOR UPDATE;

    IF p_ok THEN
        UPDATE deliveries
        SET status='done',
            status_code = p_status_code,
            response_body = p_response_body,
            last_error = NULL,
            sent_at = COALESCE(sent_at, now()),
            updated_at = now()
        WHERE id = p_delivery_id;
        RETURN;
    END IF;

    IF p_retry_after_ts IS NOT NULL THEN
        v_next := p_retry_after_ts;
    ELSE
        v_next := now() + make_interval(secs => LEAST(p_backoff_min_seconds * (2 ^ GREATEST(v_attempts, 1)),
                                                      p_backoff_max_seconds));
    END IF;

    UPDATE deliveries
    SET attempts       = v_attempts + 1,
        status         = CASE WHEN v_attempts + 1 >= v_max_attempts THEN 'failed' ELSE 'ready' END,
        status_code    = p_status_code,
        response_body= p_response_body,
        last_error     = left(p_error, 2000),
        next_run_at    = CASE WHEN v_attempts + 1 >= v_max_attempts THEN now() ELSE v_next END,
        retry_after_at = p_retry_after_ts,
        updated_at     = now()
    WHERE id = p_delivery_id;
END
$$;

-- ==============================
-- 10) Roll-up view
-- ==============================
CREATE OR REPLACE VIEW request_rollup AS
SELECT r.id,
       r.uid,
       COUNT(d.*)                                                                    AS deliveries_total,
       COUNT(*) FILTER (WHERE d.status = 'done')                                     AS deliveries_done,
       COUNT(*) FILTER (WHERE d.status = 'failed')                                   AS deliveries_failed,
       COUNT(*) FILTER (WHERE d.status IN ('ready', 'processing', 'awaiting_async')) AS deliveries_pending,
       CASE
           WHEN COUNT(*) FILTER (WHERE d.status = 'failed') > 0 THEN 'failed'
           WHEN COUNT(*) FILTER (WHERE d.status IN ('ready', 'processing', 'awaiting_async')) > 0 THEN 'processing'
           WHEN COUNT(*) FILTER (WHERE d.status = 'done') > 0
               AND COUNT(*) FILTER (WHERE d.status != 'done') = 0 THEN 'done'
           ELSE 'unknown'
           END                                                                       AS rollup_status, -- alias must be before FROM
       r.created,
       r.updated
FROM requests r
         LEFT JOIN deliveries d ON d.request_id = r.id
GROUP BY r.id, r.uid, r.created, r.updated;
