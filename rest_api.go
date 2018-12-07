package url_shortner

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

var _tokenGenerator *tokenGenerator

func RunRestApi(tg *tokenGenerator, datastore datastore, apiSecretKey string, port string) error {
	_tokenGenerator = tg

	authMiddleware, err := newAuthMiddleware(&authMiddlewareConfig{
		realm: "farmx url shortner",
		identityKey: "id",
		secretKey: apiSecretKey,
		timeout: time.Hour,
		maxRefresh: time.Hour,
	}, datastore)

	if err != nil {
		log.Fatal(err.Error())
	}

	route := gin.Default()

	route.GET("/:token", Redirect)
	route.POST("/login", authMiddleware.LoginHandler)

	v1 := route.Group("/api/v1")
	v1.Use(authMiddleware.MiddlewareFunc())
	{
		v1.POST("/register/url", RegisterUrl)
	}

	return route.Run(port)
}

func Redirect(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, _tokenGenerator.GetLongUrl(c.Param("token")))
}

type registerUrlInput struct {
	LongUrl string `json:"long_url" binding:"required"`
	CustomName string `json:"custom_name"`
}

func RegisterUrl(c *gin.Context) {
	var input registerUrlInput
	var token string
	var err error

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": err.Error()})
		return
	}

	if input.CustomName != "" {
		token, err = _tokenGenerator.NewUrlWithCustomToken(input.LongUrl, input.CustomName)
	} else {
		token = _tokenGenerator.NewUrl(input.LongUrl)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"long_url": input.LongUrl,
		"hash": token,
	})
}
