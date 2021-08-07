CREATE TABLE constraints (
  id SERIAL PRIMARY KEY,
  namespace CHARACTER VARYING(255) NOT NULL,
  name CHARACTER VARYING(255) NOT NULL,
  selector JSONB,
  allowed_processes JSONB
);

CREATE INDEX constraints_namespace_idx ON constraints (namespace);
CREATE UNIQUE INDEX constraints_namespace_name_idx ON constraints (namespace, name);

CREATE TABLE events (
  id BIGSERIAL PRIMARY KEY,
  type CHARACTER VARYING(255) NOT NULL,
  server_timestamp TIMESTAMP WITHOUT TIME ZONE NOT NULL,
  client_timestamp TIMESTAMP WITHOUT TIME ZONE NOT NULL,
  targets JSONB
);

CREATE INDEX events_type_server_timestamp_idx ON events (type,  server_timestamp);
CREATE INDEX events_server_timestamp_type_idx ON events (server_timestamp, type);
