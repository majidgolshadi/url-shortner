package url_shortner

import (
	"github.com/appleboy/gin-jwt"
	"github.com/gin-gonic/gin"
	"time"
)

type authMiddlewareConfig struct {
	realm       string
	secretKey   string
	identityKey string
	timeout     time.Duration
	maxRefresh  time.Duration
}

type login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

type User struct {
	UserName string
}

func newAuthMiddleware(cnf *authMiddlewareConfig, datastore datastore) (*jwt.GinJWTMiddleware, error) {
	return jwt.New(&jwt.GinJWTMiddleware{
		Realm:       cnf.realm,
		Key:         []byte(cnf.secretKey),
		Timeout:     cnf.timeout,
		MaxRefresh:  cnf.maxRefresh,
		IdentityKey: cnf.identityKey,

		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*User); ok {
				return jwt.MapClaims{
					cnf.identityKey: v.UserName,
				}
			}
			return jwt.MapClaims{}
		},

		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &User{
				UserName: claims["id"].(string),
			}
		},

		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginData login

			if err := c.Bind(&loginData); err != nil {
				return "", jwt.ErrMissingLoginValues
			}

			if datastore.authorizedUser(loginData.Username, loginData.Password) {
				return &User{
					UserName: loginData.Username,
				}, nil
			}

			return nil, jwt.ErrFailedAuthentication
		},

		Authorizator: func(data interface{}, c *gin.Context) bool {
			if v, ok := data.(*User); ok && v.UserName != "" {
				return true
			}

			return false
		},

		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
			})
		},
		TokenLookup:   "header: Authorization",
		TokenHeadName: "Farmx",
		TimeFunc:      time.Now,
	})
}
