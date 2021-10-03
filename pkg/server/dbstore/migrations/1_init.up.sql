CREATE TABLE constraints (
  id SERIAL PRIMARY KEY,
  namespace CHARACTER VARYING(255) NOT NULL,
  name CHARACTER VARYING(255) NOT NULL,
  selector JSONB,
  allowed_processes JSONB,
  allowed_files JSONB
);

CREATE INDEX constraints_namespace_idx ON constraints (namespace);
CREATE UNIQUE INDEX constraints_namespace_name_idx ON constraints (namespace, name);

CREATE TABLE events (
  id BIGSERIAL PRIMARY KEY,
  uid UUID NOT NULL,
  type CHARACTER VARYING(255) NOT NULL,
  server_timestamp TIMESTAMP WITHOUT TIME ZONE NOT NULL,
  client_timestamp TIMESTAMP WITHOUT TIME ZONE NOT NULL,
  alert_sent_at TIMESTAMP WITHOUT TIME ZONE NULL,
  targets JSONB
);

CREATE INDEX events_type_server_timestamp_idx ON events (type,  server_timestamp);
CREATE INDEX events_server_timestamp_type_idx ON events (server_timestamp, type);

CREATE TABLE actions (
  id SERIAL PRIMARY KEY,
  namespace CHARACTER VARYING(255) NOT NULL,
  name CHARACTER VARYING(255) NOT NULL,
  selector JSONB,
  on_violated_process boolean,
  on_violated_file boolean,
  action CHARACTER VARYING(255) NOT NULL
);

CREATE INDEX actions_namespace_idx ON actions (namespace);
CREATE UNIQUE INDEX actions_namespace_name_idx ON actions (namespace, name);
