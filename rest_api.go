package url_shortner

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func RunRestApi(port string) error {
	router := gin.Default()

	router.GET("/:token", Redirect)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/register/url", RegisterUrl)
		//v1.GET("/report/url/:token", RegisterClusterHandler)
	}

	return router.Run(port)
}

func Redirect(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, GetLongUrl(c.Param("token")))
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
		token, err = NewUrlWithCustomToken(input.LongUrl, input.CustomName)
	} else {
		token = NewUrl(input.LongUrl)
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
