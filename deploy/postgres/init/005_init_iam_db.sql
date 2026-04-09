-- Spec-08: Create dedicated database for go-iam service (FR-10)
-- go-iam uses a separate database for identity/organization data isolation (NFR-04)
SELECT 'CREATE DATABASE rcprotocol_iam'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'rcprotocol_iam')\gexec
