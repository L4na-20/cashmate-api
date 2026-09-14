-- Execute only after the new API has passed tenant and balance smoke tests.
-- GORM's historical constraint names are used by the current schema; verify
-- SHOW CREATE TABLE output before running on a customized database.

SET @wallet_user_fk = (
    SELECT CONSTRAINT_NAME
    FROM information_schema.KEY_COLUMN_USAGE
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'wallets'
      AND COLUMN_NAME = 'user_id'
      AND REFERENCED_TABLE_NAME = 'users'
    LIMIT 1
);
SET @drop_wallet_fk = IF(
    @wallet_user_fk IS NULL,
    'SELECT 1',
    CONCAT('ALTER TABLE wallets DROP FOREIGN KEY `', @wallet_user_fk, '`')
);
PREPARE stmt FROM @drop_wallet_fk;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @category_user_fk = (
    SELECT CONSTRAINT_NAME
    FROM information_schema.KEY_COLUMN_USAGE
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'categories'
      AND COLUMN_NAME = 'user_id'
      AND REFERENCED_TABLE_NAME = 'users'
    LIMIT 1
);
SET @drop_category_fk = IF(
    @category_user_fk IS NULL,
    'SELECT 1',
    CONCAT('ALTER TABLE categories DROP FOREIGN KEY `', @category_user_fk, '`')
);
PREPARE stmt FROM @drop_category_fk;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_wallet_user_column = IF(
    EXISTS (
        SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'wallets' AND COLUMN_NAME = 'user_id'
    ),
    'ALTER TABLE wallets DROP COLUMN user_id',
    'SELECT 1'
);
PREPARE stmt FROM @drop_wallet_user_column;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_category_user_column = IF(
    EXISTS (
        SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'categories' AND COLUMN_NAME = 'user_id'
    ),
    'ALTER TABLE categories DROP COLUMN user_id',
    'SELECT 1'
);
PREPARE stmt FROM @drop_category_user_column;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
