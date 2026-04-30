CREATE TABLE IF NOT EXISTS users (
	id             			 INTEGER PRIMARY KEY,
	username       			 TEXT     NOT NULL UNIQUE COLLATE NOCASE CHECK (length(trim(username)) > 0),
	password_hash  			 TEXT     NOT NULL,
	password_change_required BOOLEAN  NOT NULL DEFAULT 0,
	created_at     			 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at     			 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Update the 'updated_at' timestamp on user updates
CREATE TRIGGER IF NOT EXISTS users_set_updated_at
AFTER UPDATE OF username, password_hash, password_change_required ON users
FOR EACH ROW
WHEN NEW.updated_at IS OLD.updated_at
BEGIN
  UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

