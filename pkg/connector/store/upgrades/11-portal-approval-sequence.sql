-- v11 (compatible with v2+): Use a Postgres sequence for portal approval IDs

-- only: postgres
CREATE SEQUENCE IF NOT EXISTS telegram_portal_approval_id_seq;

-- only: postgres
ALTER TABLE telegram_portal_approval ALTER COLUMN approval_id SET DEFAULT nextval('telegram_portal_approval_id_seq');

-- only: postgres
ALTER SEQUENCE telegram_portal_approval_id_seq OWNED BY telegram_portal_approval.approval_id;

-- only: postgres
SELECT setval('telegram_portal_approval_id_seq', COALESCE((SELECT MAX(approval_id) FROM telegram_portal_approval), 0) + 1, false);
