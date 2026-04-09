-- Spec-10: Create dedicated database for go-approval service (FR-03)
-- go-approval uses a separate database for approval data isolation (NFR-02)
SELECT 'CREATE DATABASE rcprotocol_approval'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'rcprotocol_approval')\gexec
