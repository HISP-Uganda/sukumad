BEGIN;
WITH upsert AS (
    INSERT INTO user_roles (name, description)
        VALUES
            ('Admin',    'Full administrative access'),
            ('Editor', 'Can create and edit resources'),
            ('Viewer',   'Read-only access')
        ON CONFLICT (name) DO UPDATE
            SET description = EXCLUDED.description
        RETURNING id, name
)
SELECT 1;

-- 2) Permissions (idempotent)
WITH upsert AS (
    INSERT INTO permissions (name, code, system_module)
        VALUES
            ('Admin',                 'admin',                'System'),
            ('Can view requests',     'can_view_requests',    'Requests'),
            ('Can create requests',   'can_create_requests',  'Requests'),
            ('Can edit requests',     'can_edit_requests',    'Requests'),
            ('Can delete requests',   'can_delete_requests',  'Requests'),
            ('Can expand requests',   'can_expand_requests',  'Requests'),
            ('Can view users',        'can_view_users',       'Users'),
            ('Can create users',      'can_create_users',     'Users'),
            ('Can edit users',        'can_edit_users',       'Users'),
            ('Can delete users',      'can_delete_users',     'Users'),
            ('Can unlock users',      'can_user_unlock',          'Users')
        ON CONFLICT (code) DO UPDATE
            SET name = EXCLUDED.name,
                system_module = EXCLUDED.system_module
        RETURNING id, code
)
SELECT 1;

-- 3) Grant ALL permissions to Admin role (idempotent)
WITH admin_role AS (
    SELECT id AS role_id FROM user_roles WHERE name = 'Admin'
),
     missing_grants AS (
         SELECT ar.role_id, p.id AS permission_id
         FROM admin_role ar
                  CROSS JOIN permissions p
                  LEFT JOIN user_role_permissions urp
                            ON urp.role_id = ar.role_id AND urp.permission_id = p.id
         WHERE urp.role_id IS NULL
     )
INSERT INTO user_role_permissions (role_id, permission_id)
SELECT role_id, permission_id FROM missing_grants
ON CONFLICT DO NOTHING;

-- 4) Ensure demo Operator grants (optional, idempotent)
WITH ed_role AS (
    SELECT id AS role_id FROM user_roles WHERE name = 'Operator'
),
     need_grants AS (
         SELECT orl.role_id, p.id AS permission_id
         FROM ed_role orl
                  JOIN permissions p ON p.code IN ('can_view_requests', 'can_expand_requests')
                  LEFT JOIN user_role_permissions urp
                            ON urp.role_id = orl.role_id AND urp.permission_id = p.id
         WHERE urp.role_id IS NULL
     )
INSERT INTO user_role_permissions (role_id, permission_id)
SELECT role_id, permission_id FROM need_grants
ON CONFLICT DO NOTHING;

-- 5) Bootstrap users (idempotent)
-- DO NOT overwrite password on conflict; if user exists, keep current hash.
-- Admin user (system)
WITH ar AS (SELECT id FROM user_roles WHERE name = 'Admin')
INSERT INTO users (
    user_role, firstname, lastname, username, telephone, password_hash, email,
    allowed_ips, denied_ips, failed_attempts, transaction_limit,
    is_active, is_system_user, created, updated
)
SELECT
    ar.id, 'System', 'Admin', 'admin', '+256782820208',
    crypt('@dm1n', gen_salt('bf')), 'sekiskylink@gmail.com',
    NULL, NULL, 0, NULL,
    TRUE, TRUE, now(), now()
FROM ar
ON CONFLICT (username) DO NOTHING;

-- Operator user (demo)
WITH rr AS (SELECT id FROM user_roles WHERE name = 'Operator')
INSERT INTO users (
    user_role, firstname, lastname, username, telephone, password_hash, email,
    allowed_ips, denied_ips, failed_attempts, transaction_limit,
    is_active, is_system_user, created, updated
)
SELECT
    rr.id, 'Editor', 'Editor', 'editor', '+256700000001',
    crypt('Operator!123', gen_salt('bf')), 'operator@example.org',
    NULL, NULL, 0, NULL,
    TRUE, FALSE, now(), now()
FROM rr
ON CONFLICT (username) DO NOTHING;

COMMIT ;