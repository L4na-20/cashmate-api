package models

import "strings"

const (
	RoleOwner = "OWNER"
	RoleStaff = "STAFF"

	TransactionIncome  = "income"
	TransactionExpense = "expense"
	CurrencyIDR        = "IDR"
)

// NormalizeRole keeps legacy lowercase data readable while all new data uses
// the API contract's uppercase role values.
func NormalizeRole(role string) string {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case RoleOwner:
		return RoleOwner
	case RoleStaff:
		return RoleStaff
	default:
		return ""
	}
}
