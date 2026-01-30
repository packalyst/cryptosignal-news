package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/cryptosignal-news/backend/internal/models"
)

var (
	// ErrInvalidToken is returned when a token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when a token has expired
	ErrExpiredToken = errors.New("token has expired")
	// ErrTokenNotYetValid is returned when a token is not yet valid
	ErrTokenNotYetValid = errors.New("token is not yet valid")
)

// Claims represents the JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Tier   string `json:"tier"`
	jwt.RegisteredClaims
}

// JWTService handles JWT token operations
type JWTService struct {
	secret     []byte
	expiration time.Duration
	issuer     string
}

// NewJWTService creates a new JWT service
func NewJWTService(secret string, expiration time.Duration) *JWTService {
	return &JWTService{
		secret:     []byte(secret),
		expiration: expiration,
		issuer:     "cryptosignal-news",
	}
}

// Generate creates a new JWT token for a user
func (s *JWTService) Generate(user *models.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		Tier:   user.Tier,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// Validate validates a JWT token and returns the claims
func (s *JWTService) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenNotYetValid
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Additional validation
	if claims.Issuer != s.issuer {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// Refresh refreshes a JWT token (creates a new token with extended expiration)
func (s *JWTService) Refresh(tokenString string) (string, error) {
	claims, err := s.Validate(tokenString)
	if err != nil {
		// Allow refresh of expired tokens within a grace period (e.g., 7 days)
		if err == ErrExpiredToken {
			token, parseErr := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				return s.secret, nil
			})
			if parseErr != nil {
				return "", ErrInvalidToken
			}

			claims, ok := token.Claims.(*Claims)
			if !ok {
				return "", ErrInvalidToken
			}

			// Check grace period (7 days after expiration)
			if claims.ExpiresAt != nil {
				gracePeriod := 7 * 24 * time.Hour
				if time.Since(claims.ExpiresAt.Time) > gracePeriod {
					return "", ErrExpiredToken
				}
			}

			// Create new token
			return s.generateFromClaims(claims)
		}
		return "", err
	}

	return s.generateFromClaims(claims)
}

// generateFromClaims creates a new token from existing claims
func (s *JWTService) generateFromClaims(oldClaims *Claims) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: oldClaims.UserID,
		Email:  oldClaims.Email,
		Tier:   oldClaims.Tier,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   oldClaims.UserID,
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// GetExpiration returns the token expiration duration
func (s *JWTService) GetExpiration() time.Duration {
	return s.expiration
}
