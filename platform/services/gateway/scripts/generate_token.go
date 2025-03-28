package main

import (
    "fmt"
    "os"
    "time"

    "github.com/golang-jwt/jwt/v4"
)

type CustomClaims struct {
    UserID string   `json:"user_id"`
    Roles  []string `json:"roles"`
    jwt.RegisteredClaims
}

func main() {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        secret = "mysecret"
    }

    userID := "demo-user"
    roles := []string{"user"}

    exp := time.Now().Add(15 * time.Minute)

    claims := CustomClaims{
        UserID: userID,
        Roles:  roles,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(exp),
            Issuer:    "local-script",
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

    signed, err := token.SignedString([]byte(secret))
    if err != nil {
        panic(err)
    }

    fmt.Printf("JWT Token for user=%s, roles=%v (expires in 15m):\n", userID, roles)
    fmt.Println(signed)
}

