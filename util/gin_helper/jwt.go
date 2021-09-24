package gin_helper

import (
	"gitee.com/kelvins-io/kelvins"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"net/http"
	"time"
)

func CheckUserToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("token")
		if token == "" {
			JsonResponse(c, http.StatusUnauthorized, ErrorTokenEmpty, GetMsg(ErrorTokenEmpty))
			c.Abort()
			return
		}
		claims, err := ParseToken(token)
		if err != nil {
			JsonResponse(c, http.StatusForbidden, ErrorTokenInvalid, GetMsg(ErrorTokenInvalid))
			c.Abort()
			return
		} else if claims == nil || claims.Uid == 0 {
			JsonResponse(c, http.StatusForbidden, ErrorUserNotExist, GetMsg(ErrorUserNotExist))
			c.Abort()
			return
		} else if time.Now().Unix() > claims.ExpiresAt {
			JsonResponse(c, http.StatusForbidden, ErrorTokenExpire, GetMsg(ErrorTokenExpire))
			c.Abort()
			return
		}

		c.Set("uid", claims.Uid)
		c.Next()
	}
}

const (
	jwtSecret     = "&WJof0jaY4ByTHR2"
	jwtExpireTime = 2 * time.Hour
)

type Claims struct {
	UserName string `json:"user_name"`
	Uid      int    `json:"uid"`
	jwt.StandardClaims
}

func GenerateToken(username string, uid int) (string, error) {
	var expire = jwtExpireTime
	if kelvins.JwtSetting != nil && kelvins.JwtSetting.TokenExpireSecond > 0 {
		expire = time.Duration(kelvins.JwtSetting.TokenExpireSecond) * time.Second
	}
	var secret = jwtSecret
	if kelvins.JwtSetting != nil && kelvins.JwtSetting.Secret != "" {
		secret = kelvins.JwtSetting.Secret
	}
	nowTime := time.Now()
	expireTime := nowTime.Add(expire)

	claims := Claims{
		UserName: username,
		Uid:      uid,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    "kelvins-io/kelvins",
		},
	}
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	token, err := tokenClaims.SignedString([]byte(secret))
	return token, err
}

func ParseToken(token string) (*Claims, error) {
	var secret = jwtSecret
	if kelvins.JwtSetting != nil && kelvins.JwtSetting.Secret != "" {
		secret = kelvins.JwtSetting.Secret
	}
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (i interface{}, err error) {
		return []byte(secret), nil
	})
	if tokenClaims != nil {
		if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
			return claims, nil
		}
	}
	return nil, err
}
