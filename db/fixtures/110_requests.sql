-- Create a couple of requests; rely on server names to find IDs
WITH dhis AS (SELECT id FROM servers WHERE name='dhis2_hmis'),
     hook AS (SELECT id FROM servers WHERE name='dhis2_hmis_local'),
INSERT INTO requests (uid, source, destination, cc_servers, depends_on, batchid,
                      url_suffix, body, body_is_query_param,
                      frequency_type, period, year, report_type,
                      idempotency_key, suspended, created, updated)
VALUES
    (gen_random_uuid()::text, NULL, (SELECT id FROM dhis), ARRAY[(SELECT id FROM hook)], NULL, 'batch-001',
     NULL, '{"dataSet":"abc123","dataValues":[]}', false,
     'monthly', '202507', 2025, 'dataValueSet',
     gen_random_uuid()::text, false, now(), now()),
    (gen_random_uuid()::text, NULL, (SELECT id FROM hook), NULL, NULL, 'batch-002',
     '/daily', 'key=value&x=1', true,
     'daily', '2025-08-16', 2025, 'webhook',
     gen_random_uuid()::text, false, now(), now())
ON CONFLICT DO NOTHING;
