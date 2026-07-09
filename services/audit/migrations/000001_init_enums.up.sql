-- Create enums
CREATE TYPE actor_type AS ENUM ('user', 'api_key', 'system', 'anonymous');
CREATE TYPE event_result AS ENUM ('success', 'failure', 'denied');
