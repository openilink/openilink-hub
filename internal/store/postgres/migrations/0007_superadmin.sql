-- +goose Up

-- Promote the earliest admin to superadmin
UPDATE users SET role = 'superadmin'
WHERE id = (SELECT id FROM users WHERE role = 'admin' ORDER BY created_at ASC LIMIT 1)
  AND NOT EXISTS (SELECT 1 FROM users WHERE role = 'superadmin');
