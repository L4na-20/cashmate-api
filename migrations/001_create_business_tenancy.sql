-- CashMate tenant expansion. Run against a backup/clone first.
-- This migration is intentionally explicit; production startup must not
-- silently mutate the schema.

CREATE TABLE IF NOT EXISTS businesses (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    INDEX idx_businesses_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE users
    ADD COLUMN business_id BIGINT UNSIGNED NULL,
    ADD COLUMN auth_version BIGINT UNSIGNED NOT NULL DEFAULT 0,
    ADD INDEX idx_users_business_role (business_id, role, deleted_at);

ALTER TABLE wallets
    ADD COLUMN business_id BIGINT UNSIGNED NULL,
    ADD INDEX idx_wallets_business_deleted (business_id, deleted_at);

ALTER TABLE categories
    ADD COLUMN business_id BIGINT UNSIGNED NULL,
    ADD INDEX idx_categories_business_type (business_id, type, deleted_at);

ALTER TABLE transactions
    ADD COLUMN business_id BIGINT UNSIGNED NULL,
    ADD COLUMN created_by_user_id BIGINT UNSIGNED NULL,
    ADD COLUMN updated_by_user_id BIGINT UNSIGNED NULL,
    ADD COLUMN deleted_by_user_id BIGINT UNSIGNED NULL,
    ADD INDEX idx_transactions_business_date (business_id, date, deleted_at),
    ADD INDEX idx_transactions_creator_date (created_by_user_id, date, deleted_at);
