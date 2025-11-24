package validator

import "time"

// IsValidDate valida que la fecha tenga formato YYYY-MM-DD
func IsValidDate(date string) bool {
    if len(date) != 10 {
        return false
    }
    _, err := time.Parse("2006-01-02", date)
    return err == nil
}
