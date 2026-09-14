-- Legacy mapping policy:
-- the previous schema had one global Owner and user-owned wallets, so all
-- existing users are placed in one tenant. Review row counts before running.

-- Abort manually if the preflight does not report exactly one active owner.
-- SELECT role, COUNT(*) FROM users WHERE deleted_at IS NULL GROUP BY role;

SET @legacy_business_id = NULL;

INSERT INTO businesses (name, created_at, updated_at)
SELECT 'CashMate Legacy', NOW(3), NOW(3)
WHERE EXISTS (SELECT 1 FROM users LIMIT 1);

SET @legacy_business_id = LAST_INSERT_ID();

UPDATE users
SET business_id = @legacy_business_id
WHERE business_id IS NULL;

UPDATE users
SET role = UPPER(role)
WHERE role IS NOT NULL;

UPDATE wallets w
JOIN users u ON u.id = w.user_id
SET w.business_id = u.business_id
WHERE w.business_id IS NULL;

UPDATE categories c
JOIN users u ON u.id = c.user_id
SET c.business_id = u.business_id
WHERE c.business_id IS NULL AND c.user_id IS NOT NULL;

UPDATE transactions t
JOIN wallets w ON w.id = t.wallet_id
SET t.business_id = w.business_id,
    t.created_by_user_id = COALESCE(t.created_by_user_id, w.user_id)
WHERE t.business_id IS NULL;

-- Existing wallets were allowed to be edited directly. Reconcile any
-- discrepancy with explicit ledger entries before converting money to integer
-- rupiah. Run the query below first and stop if any amount has a fractional
-- rupiah; the BIGINT migration deliberately never rounds money.
-- SELECT w.id, w.balance,
--        COALESCE(SUM(CASE WHEN t.type='income' THEN t.amount ELSE -t.amount END), 0) AS ledger_total
-- FROM wallets w LEFT JOIN transactions t ON t.wallet_id=w.id AND t.deleted_at IS NULL
-- GROUP BY w.id, w.balance HAVING w.balance <> ledger_total;

INSERT INTO categories (business_id, name, type, created_at, updated_at)
SELECT @legacy_business_id, 'Saldo Awal Migrasi', 'income', NOW(3), NOW(3)
WHERE @legacy_business_id IS NOT NULL AND @legacy_business_id > 0
  AND NOT EXISTS (
    SELECT 1 FROM categories
    WHERE business_id = @legacy_business_id AND name = 'Saldo Awal Migrasi' AND type = 'income'
);

INSERT INTO categories (business_id, name, type, created_at, updated_at)
SELECT @legacy_business_id, 'Penyesuaian Saldo Migrasi', 'expense', NOW(3), NOW(3)
WHERE @legacy_business_id IS NOT NULL AND @legacy_business_id > 0
  AND NOT EXISTS (
    SELECT 1 FROM categories
    WHERE business_id = @legacy_business_id AND name = 'Penyesuaian Saldo Migrasi' AND type = 'expense'
);

INSERT INTO transactions (
    business_id, wallet_id, category_id, created_by_user_id, amount, type,
    description, date, created_at, updated_at
)
SELECT
    w.business_id,
    w.id,
    CASE WHEN w.balance - COALESCE(ledger.ledger_total, 0) >= 0 THEN income_category.id ELSE expense_category.id END,
    w.user_id,
    ABS(w.balance - COALESCE(ledger.ledger_total, 0)),
    CASE WHEN w.balance - COALESCE(ledger.ledger_total, 0) >= 0 THEN 'income' ELSE 'expense' END,
    'Migrated opening balance reconciliation',
    COALESCE(DATE(w.created_at), CURRENT_DATE),
    NOW(3),
    NOW(3)
FROM wallets w
LEFT JOIN (
    SELECT wallet_id,
           COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0) AS ledger_total
    FROM transactions
    WHERE deleted_at IS NULL
    GROUP BY wallet_id
) ledger ON ledger.wallet_id = w.id
JOIN categories income_category
    ON income_category.business_id = w.business_id
    AND income_category.name = 'Saldo Awal Migrasi'
    AND income_category.type = 'income'
JOIN categories expense_category
    ON expense_category.business_id = w.business_id
    AND expense_category.name = 'Penyesuaian Saldo Migrasi'
    AND expense_category.type = 'expense'
WHERE w.business_id = @legacy_business_id
  AND w.balance <> COALESCE(ledger.ledger_total, 0)
  AND NOT EXISTS (
      SELECT 1 FROM transactions existing_adjustment
      WHERE existing_adjustment.wallet_id = w.id
        AND existing_adjustment.description = 'Migrated opening balance reconciliation'
        AND existing_adjustment.deleted_at IS NULL
  );
