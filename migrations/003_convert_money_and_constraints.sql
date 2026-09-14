-- Run only after the preflight confirms every amount/balance is a whole rupiah
-- and no business/creator columns are NULL.

ALTER TABLE users
    MODIFY COLUMN business_id BIGINT UNSIGNED NOT NULL,
    MODIFY COLUMN role VARCHAR(20) NOT NULL DEFAULT 'STAFF';

ALTER TABLE wallets
    MODIFY COLUMN business_id BIGINT UNSIGNED NOT NULL,
    MODIFY COLUMN balance BIGINT NOT NULL DEFAULT 0,
    MODIFY COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'IDR';

ALTER TABLE categories
    MODIFY COLUMN business_id BIGINT UNSIGNED NULL;

ALTER TABLE transactions
    MODIFY COLUMN business_id BIGINT UNSIGNED NOT NULL,
    MODIFY COLUMN created_by_user_id BIGINT UNSIGNED NOT NULL,
    MODIFY COLUMN amount BIGINT UNSIGNED NOT NULL;

ALTER TABLE users
    ADD CONSTRAINT fk_users_business FOREIGN KEY (business_id) REFERENCES businesses(id)
        ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE wallets
    ADD CONSTRAINT fk_wallets_business FOREIGN KEY (business_id) REFERENCES businesses(id)
        ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE categories
    ADD CONSTRAINT fk_categories_business FOREIGN KEY (business_id) REFERENCES businesses(id)
        ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE transactions
    ADD CONSTRAINT fk_transactions_business FOREIGN KEY (business_id) REFERENCES businesses(id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    ADD CONSTRAINT fk_transactions_created_by FOREIGN KEY (created_by_user_id) REFERENCES users(id)
        ON UPDATE CASCADE ON DELETE RESTRICT;
