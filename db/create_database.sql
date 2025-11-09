-- Run this script from a superuser (e.g. postgres) to create the Shipman database
-- and ensure the local role `demetrigeras` owns it.

DO
$$
BEGIN
    IF NOT EXISTS (
        SELECT FROM pg_catalog.pg_roles WHERE rolname = 'demetrigeras'
    ) THEN
        RAISE EXCEPTION 'Role "demetrigeras" does not exist. Create it before running this script.';
    END IF;
END;
$$;

\echo Creating database shipman (ignored if it already exists)
SELECT 'CREATE DATABASE shipman OWNER demetrigeras ENCODING ''UTF8'' TEMPLATE template0'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'shipman')
\gexec

ALTER DATABASE shipman OWNER TO demetrigeras;

