package token

import (
	"fmt"
	"time"

	"github.com/pranav244872/synapse/db/sqlc"

	// The official Go JWT library for working with JSON Web Tokens.
	"github.com/golang-jwt/jwt/v5"
)

// JWTMaker is a struct that handles creation and verification of JWT tokens.
type JWTMaker struct {
	secretKey string // A secret key used to sign and verify JWTs.
}

// NewJWTMaker creates a new JWTMaker with the provided secret key.
// The key must be at least 32 characters long to ensure strong encryption.
func NewJWTMaker(secretKey string) (*JWTMaker, error) {
	if len(secretKey) < 32 {
		// If the key is too short, return an error.
		return nil, fmt.Errorf("invalid key size: must be at least 32 characters")
	}
	// Return a pointer to the new JWTMaker with the given secret key.
	return &JWTMaker{secretKey}, nil
}

// CreateToken generates a JWT token for a specific user.
// Parameters:
// - userID: the ID of the user
// - role: the user's role (from your database)
// - duration: how long the token will be valid
func (maker *JWTMaker) CreateToken(userID int64, role db.UserRole, duration time.Duration) (string, error) {
	// Define the payload (data stored inside the token)
	payload := jwt.MapClaims{
		"user_id": userID,                        // Custom claim: the user's ID
		"role":    role,                          // Custom claim: the user's role
		"exp":     time.Now().Add(duration).Unix(), // Standard claim: expiration time
		"iat":     time.Now().Unix(),               // Standard claim: issued at time
	}

	// Create a new JWT token using the HS256 signing algorithm
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	// Sign the token with the secret key and return it
	return jwtToken.SignedString([]byte(maker.secretKey))
}

// VerifyToken checks if the given JWT token is valid and not expired.
// If valid, it returns the claims (the payload inside the token).
func (maker *JWTMaker) VerifyToken(tokenString string) (jwt.MapClaims, error) {
	// Parse the token and provide a function to supply the secret key for verification
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Check if the signing method is HMAC (like HS256)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			// If not, reject the token
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Return the secret key used to verify the signature
		return []byte(maker.secretKey), nil
	})

	// If there's an error in parsing (e.g., invalid token), return it
	if err != nil {
		return nil, err
	}

	// Convert the claims to a map (key-value format)
	claims, ok := token.Claims.(jwt.MapClaims)
	// Also check if the token is valid
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Return the claims from the token (like user_id, role, etc.)
	return claims, nil
}
