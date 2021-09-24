package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	grpcAuth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"strconv"
	"strings"
	"time"
)

var (
	tokenValidityDurationDefault = 30 * time.Second
	errUnauthenticated           = status.Errorf(codes.Unauthenticated, "authentication required")
	errDenied                    = status.Errorf(codes.PermissionDenied, "permission denied")
)

func RPCPerCredentials(sharedSecret string) credentials.PerRPCCredentials {
	return &rpcPerAuthCredentials{sharedSecret: sharedSecret}
}

// GetRPCPerAuthHeader 根据私钥获取RPC 接入auth header
func GetRPCPerAuthHeader(secret string) (map[string]string, error) {
	x := rpcPerAuthCredentials{
		sharedSecret: secret,
	}
	return x.GetRequestMetadata(context.Background())
}

type rpcPerAuthCredentials struct {
	sharedSecret string
}

func (*rpcPerAuthCredentials) RequireTransportSecurity() bool { return false }

func (rc *rpcPerAuthCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	message := strconv.FormatInt(time.Now().Unix(), 10)
	signature := hmacSign([]byte(rc.sharedSecret), message)

	return map[string]string{
		"authorization": "Bearer " + fmt.Sprintf("v1.%x.%s", signature, message),
	}, nil
}

func hmacSign(secret []byte, message string) []byte {
	mac := hmac.New(sha256.New, secret)
	// hash.Hash never returns an error.
	_, _ = mac.Write([]byte(message))

	return mac.Sum(nil)
}

func hmacInfoValid(message string, signedMessage, secret []byte, targetTime time.Time, tokenValidity time.Duration) bool {
	expectedHMAC := hmacSign(secret, message)
	if !hmac.Equal(signedMessage, expectedHMAC) {
		return false
	}

	timestamp, err := strconv.ParseInt(message, 10, 64)
	if err != nil {
		return false
	}

	issuedAt := time.Unix(timestamp, 0)
	lowerBound := targetTime.Add(-tokenValidity)
	upperBound := targetTime.Add(tokenValidity)

	if issuedAt.Before(lowerBound) {
		return false
	}

	if issuedAt.After(upperBound) {
		return false
	}

	return true
}

type AuthInfo struct {
	Version       string
	SignedMessage []byte
	Message       string
}

func checkToken(ctx context.Context, secret string, targetTime time.Time, tokenValidityDuration time.Duration) (*AuthInfo, error) {
	if len(secret) == 0 {
		panic("checkToken: secret may not be empty")
	}
	if tokenValidityDuration < 0 {
		tokenValidityDuration = tokenValidityDurationDefault
	}

	authInfo, err := extractAuthInfo(ctx)
	if err != nil {
		return nil, errUnauthenticated
	}

	if authInfo.Version == "v1" {
		if hmacInfoValid(authInfo.Message, authInfo.SignedMessage, []byte(secret), targetTime, tokenValidityDuration) {
			return authInfo, nil
		}
	}

	return nil, errDenied
}

func extractAuthInfo(ctx context.Context) (*AuthInfo, error) {
	token, err := grpcAuth.AuthFromMD(ctx, "bearer")

	if err != nil {
		return nil, err
	}

	split := strings.SplitN(token, ".", 3)

	if len(split) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	version, sig, msg := split[0], split[1], split[2]
	decodedSig, err := hex.DecodeString(sig)
	if err != nil {
		return nil, err
	}

	return &AuthInfo{Version: version, SignedMessage: decodedSig, Message: msg}, nil
}
