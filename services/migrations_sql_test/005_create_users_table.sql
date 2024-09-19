-- Create the users table with constraints directly
CREATE TABLE users (
  id UUID PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  address VARCHAR(255),
  phone VARCHAR(20),
  profile_picture_url VARCHAR(255),
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  role VARCHAR(50) NOT NULL,
  CONSTRAINT users_role_check CHECK (role IN ('standard_user', 'organization_user', 'suborganization_user'))
);

-- Create an index on the email column for better query performance
CREATE INDEX users_email_index ON users (email);

