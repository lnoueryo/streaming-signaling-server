package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type UserInfo struct {
    ID    string `json:"id"`
    Email string `json:"email"`
    Name  string `json:"name"`
    Image  string `json:"image"`
}

var firebaseApp *firebase.App
var firebaseAuth *auth.Client

func InitFirebase() {
    opt := option.WithCredentialsFile(".credentials/firebase-admin.json")
    config := &firebase.Config{ProjectID: "streaming-cb408"}

    app, err := firebase.NewApp(context.Background(), config, opt)
    if err != nil {
        log.Fatalf("Failed to init Firebase: %v", err)
    }
    firebaseApp = app

    authClient, err := app.Auth(context.Background())
    if err != nil {
        log.Fatalf("Failed to init Firebase Auth: %v", err)
    }
    firebaseAuth = authClient
}

func VerifyIDToken(idToken string) (*auth.Token, error) {
    return firebaseAuth.VerifyIDToken(context.Background(), idToken)
}

func VerifySessionCookieAndCheckRevoked(sessionCookie string) (*auth.Token, error) {
    return firebaseAuth.VerifySessionCookieAndCheckRevoked(context.Background(), sessionCookie)
}

func FirebaseWebsocketAuth() gin.HandlerFunc {
    return func(c *gin.Context) {

        session, _ := c.Cookie("session")
        if session == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "トークンがありません"})
            return
        }

        // 2) Firebase で検証
        token, err := VerifySessionCookieAndCheckRevoked(session)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "無効なトークンです"})
            return
        }


        uid := token.UID
        email, _ := token.Claims["email"].(string)
        name, _ := token.Claims["name"].(string)
        image, _ := token.Claims["picture"].(string)

        c.Set("user", UserInfo{
            ID:   uid,
            Name:  name,
            Email: email,
            Image: image,
        })

        c.Next()
    }
}

func AuthHttpInterceptor() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization missing"})
            return
        }

        tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))

        log.Info("Verifying token:", tokenString)
        claims, err := verifyServiceJWT(tokenString)
        if err != nil {
            log.Error("JWT verify failed:", err)
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token: " + err.Error()})
            return
        }

        c.Set("serviceClaims", claims)
        c.Next()
    }
}


func AuthGrpcInterceptor(
    ctx context.Context,
    req interface{},
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (interface{}, error) {

    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "metadata missing")
    }

    auth := md.Get("authorization")
    if len(auth) == 0 {
        return nil, status.Error(codes.Unauthenticated, "authorization missing")
    }

    token := strings.TrimPrefix(auth[0], "Bearer ")

    claims, err := verifyServiceJWT(token)
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, "invalid token")
    }

    ctx = context.WithValue(ctx, "serviceClaims", claims)
    return handler(ctx, req)
}

func verifyServiceJWT(tokenString string) (*jwt.RegisteredClaims, error) {
    secret := []byte(os.Getenv("SERVICE_JWT_SECRET"))
    token, err := jwt.ParseWithClaims(
        tokenString,
        &jwt.RegisteredClaims{},
        func(token *jwt.Token) (interface{}, error) {
            return secret, nil
        },
        jwt.WithAudience("signaling-server"),
        jwt.WithIssuer("app-server"),
    )
    if err != nil {
        return nil, fmt.Errorf("parse error: %w", err)
    }

    claims, ok := token.Claims.(*jwt.RegisteredClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid claims or token")
    }
    return claims, nil
}