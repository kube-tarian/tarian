CREATE TABLE constraints (
  id SERIAL PRIMARY KEY,
  namespace CHARACTER VARYING(255) NOT NULL,
  selector JSONB,
  allowed_processes JSONB
);

CREATE INDEX constraints_namespace_idx ON constraints (namespace);
