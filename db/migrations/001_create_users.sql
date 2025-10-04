CREATE TABLE IF NOT EXISTS users (
	id             			INTEGER PRIMARY KEY,
	username       			TEXT     NOT NULL UNIQUE COLLATE NOCASE CHECK (length(trim(username)) > 0),
	password_hash  			TEXT     NOT NULL,
	password_reset_required BOOLEAN  NOT NULL DEFAULT 0,
	created_at     			DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at     			DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Update the 'updated_at' timestamp on user updates
CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
WHEN NEW.updated_at IS OLD.updated_at
BEGIN
  SELECT NEW.updated_at = CURRENT_TIMESTAMP;
END;

-- Create a default admin user with a secure password hash if it doesn't exist
-- Default password is 'placeholder'. User is forced to change it on first login.
INSERT INTO users (username, password_hash, password_reset_required)
VALUES ('admin', '$argon2id$v=19$m=65536,t=1,p=4,l=32$weMSjfxU6+aXx8ylew5tAQ$oUu4uP4YwqXbDktayQKfj/mxKmR9fTUbghuIIReRaRA', 1)
ON CONFLICT(username) DO NOTHING;