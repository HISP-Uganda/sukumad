-- Idempotent upsert for servers
INSERT INTO servers (name, url, http_method, auth_method, default_content_type, rps, burst, max_concurrency, timeout_ms,
                     rate_class, suspended, created, updated)
VALUES ('dhis2_hmis', 'https://hmis.health.go.ug/api/dataValueSets', 'POST', 'basic', 'application/json', 10, 20, 4,
        10000, 'server:dhis2', false, now(), now()),
       ('dhis2_hmis_local', 'http://localhost:8181/api/dataValueSets', 'POST', 'none', 'application/json', 50, 100, 8, 5000,
        'server:dhis2_hmis_local', false, now(), now())
ON CONFLICT (name) DO UPDATE SET url                  = EXCLUDED.url,
                                 http_method          = EXCLUDED.http_method,
                                 auth_method          = EXCLUDED.auth_method,
                                 default_content_type = EXCLUDED.default_content_type,
                                 rps                  = EXCLUDED.rps,
                                 burst                = EXCLUDED.burst,
                                 max_concurrency      = EXCLUDED.max_concurrency,
                                 timeout_ms           = EXCLUDED.timeout_ms,
                                 rate_class           = EXCLUDED.rate_class,
                                 suspended            = EXCLUDED.suspended,
                                 updated              = now();
