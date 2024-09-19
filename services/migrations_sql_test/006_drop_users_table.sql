-- Drop the index before dropping the table
DROP INDEX IF EXISTS users_email_index;

-- Drop the users table
DROP TABLE IF EXISTS users;

