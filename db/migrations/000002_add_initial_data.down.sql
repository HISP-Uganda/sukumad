BEGIN;

-- Revoke role-permissions we added (safe to leave if you prefer)
-- This deletes ONLY grants where role is Admin/Operator and permission code is in our seeded set.
WITH target_roles AS (
    SELECT id FROM user_roles WHERE name IN ('Admin','Editor')
),
     target_perms AS (
         SELECT id FROM permissions WHERE code IN (
                'admin',
                'can_view_requests','can_create_requests','can_edit_requests',
                'can_delete_requests','can_expand_requests',
                'can_view_users','can_create_users','can_edit_users',
                'can_delete_users','can_user_unlock'
        )
     )
DELETE FROM user_role_permissions urp
    USING target_roles r, target_perms p
WHERE urp.role_id = r.id AND urp.permission_id = p.id;

-- Delete demo users we created (do not touch others)
DELETE FROM users WHERE username IN ('editor')
                    AND is_system_user = FALSE;

COMMIT;