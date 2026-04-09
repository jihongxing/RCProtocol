SELECT 'CREATE DATABASE rcprotocol_workorder'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'rcprotocol_workorder')\gexec
