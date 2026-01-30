package auth

import (
	"errors"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLength is the minimum required password length
	MinPasswordLength = 8
	// MaxPasswordLength is the maximum allowed password length (bcrypt limit)
	MaxPasswordLength = 72
	// DefaultBcryptCost is the default bcrypt cost factor
	DefaultBcryptCost = 12
)

var (
	// ErrPasswordTooShort is returned when password is too short
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	// ErrPasswordTooLong is returned when password is too long
	ErrPasswordTooLong = errors.New("password must be at most 72 characters")
	// ErrPasswordNoUpper is returned when password has no uppercase letter
	ErrPasswordNoUpper = errors.New("password must contain at least one uppercase letter")
	// ErrPasswordNoLower is returned when password has no lowercase letter
	ErrPasswordNoLower = errors.New("password must contain at least one lowercase letter")
	// ErrPasswordNoDigit is returned when password has no digit
	ErrPasswordNoDigit = errors.New("password must contain at least one digit")
	// ErrPasswordCommon is returned when password is too common
	ErrPasswordCommon = errors.New("password is too common")
)

// Common passwords to reject (partial list)
var commonPasswords = map[string]bool{
	"password":   true,
	"12345678":   true,
	"123456789":  true,
	"1234567890": true,
	"qwerty":     true,
	"qwertyuiop": true,
	"password1":  true,
	"password123": true,
	"letmein":    true,
	"welcome":    true,
	"admin":      true,
	"monkey":     true,
	"dragon":     true,
	"master":     true,
	"login":      true,
	"abc123":     true,
	"iloveyou":   true,
	"sunshine":   true,
	"princess":   true,
	"football":   true,
	"baseball":   true,
	"soccer":     true,
	"hockey":     true,
	"bitcoin":    true,
	"crypto":     true,
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultBcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword compares a password with its hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePasswordStrength validates password meets security requirements
func ValidatePasswordStrength(password string) error {
	// Check length
	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}

	// Check for common passwords
	if commonPasswords[strings.ToLower(password)] {
		return ErrPasswordCommon
	}

	var (
		hasUpper bool
		hasLower bool
		hasDigit bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}

	if !hasUpper {
		return ErrPasswordNoUpper
	}
	if !hasLower {
		return ErrPasswordNoLower
	}
	if !hasDigit {
		return ErrPasswordNoDigit
	}

	return nil
}

// PasswordStrength returns a score for password strength (0-100)
func PasswordStrength(password string) int {
	score := 0

	// Length score (up to 25 points)
	length := len(password)
	if length >= 8 {
		score += 10
	}
	if length >= 12 {
		score += 10
	}
	if length >= 16 {
		score += 5
	}

	// Character variety (up to 40 points)
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if hasUpper {
		score += 10
	}
	if hasLower {
		score += 10
	}
	if hasDigit {
		score += 10
	}
	if hasSpecial {
		score += 10
	}

	// Penalty for common patterns (up to -25 points)
	lowerPwd := strings.ToLower(password)
	if commonPasswords[lowerPwd] {
		score -= 25
	}

	// Check for sequential characters
	if containsSequential(password) {
		score -= 10
	}

	// Bonus for mixed character types throughout (up to 35 points)
	mixedCount := 0
	for i := 1; i < len(password); i++ {
		currType := charType(rune(password[i]))
		prevType := charType(rune(password[i-1]))
		if currType != prevType {
			mixedCount++
		}
	}
	if mixedCount >= 3 {
		score += 15
	}
	if mixedCount >= 5 {
		score += 10
	}
	if mixedCount >= 7 {
		score += 10
	}

	// Clamp score
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

func charType(r rune) int {
	switch {
	case unicode.IsUpper(r):
		return 1
	case unicode.IsLower(r):
		return 2
	case unicode.IsDigit(r):
		return 3
	default:
		return 4
	}
}

func containsSequential(password string) bool {
	sequences := []string{
		"123", "234", "345", "456", "567", "678", "789", "890",
		"abc", "bcd", "cde", "def", "efg", "fgh", "ghi", "hij",
		"qwe", "wer", "ert", "rty", "tyu", "yui", "uio", "iop",
		"asd", "sdf", "dfg", "fgh", "ghj", "hjk", "jkl",
		"zxc", "xcv", "cvb", "vbn", "bnm",
	}

	lowerPwd := strings.ToLower(password)
	for _, seq := range sequences {
		if strings.Contains(lowerPwd, seq) {
			return true
		}
	}
	return false
}
