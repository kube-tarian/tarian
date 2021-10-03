DROP INDEX IF EXISTS constraints_namespace_idx;
DROP INDEX IF EXISTS constraints_namespace_name_idx;
DROP TABLE IF EXISTS constraints;

DROP INDEX IF EXISTS events_type_server_timestamp_idx;
DROP INDEX IF EXISTS events_server_timestamp_type_idx;
DROP TABLE IF EXISTS events;

DROP INDEX IF EXISTS actions_namespace_idx;
DROP INDEX IF EXISTS actions_namespace_name_idx;
DROP TABLE IF EXISTS actions;
