-- Runs once at container creation (postgres image convention: everything
-- in /docker-entrypoint-initdb.d/ executes in filename order, via psql).
--
-- dbwatch_test_production already exists (POSTGRES_DB). This creates the
-- rest of the fleet dbwatch-test monitors, plus a restricted role used by
-- the permission-error scenario.

CREATE DATABASE dbwatch_test_staging;
CREATE DATABASE dbwatch_test_analytics;
CREATE DATABASE dbwatch_test_development;
CREATE DATABASE dbwatch_test_stress;

-- Deliberately weak/no password: this is a throwaway local container never
-- exposed beyond localhost, and the role has no login-worthy privileges of
-- its own beyond what each scenario grants it.
CREATE ROLE dbwatch_test_restricted LOGIN PASSWORD 'dbwatch';
