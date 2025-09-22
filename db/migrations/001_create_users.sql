CREATE TABLE IF NOT EXISTS users (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	username       TEXT     NOT NULL UNIQUE,
	password_hash  TEXT     NOT NULL,
	created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create a default admin user with a secure password hash if it doesn't exist
-- Default password is 'placeholder'. User is forced to change it on first login.
INSERT INTO users (username, password_hash) VALUES ('admin', '$argon2id$v=19$m=65536,t=1,p=4,l=32$weMSjfxU6+aXx8ylew5tAQ$oUu4uP4YwqXbDktayQKfj/mxKmR9fTUbghuIIReRaRA') ON CONFLICT(username) DO NOTHING;