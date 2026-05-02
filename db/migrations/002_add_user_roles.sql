ALTER TABLE users ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT 1;

DROP TRIGGER IF EXISTS users_set_updated_at;

CREATE TRIGGER IF NOT EXISTS users_set_updated_at
AFTER UPDATE OF username, password_hash, password_change_required, is_admin ON users
FOR EACH ROW
WHEN NEW.updated_at IS OLD.updated_at
BEGIN
  UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
