package middleware

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func JWKS() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := getClaims(c.GetHeader("Authorization"))
		if err != nil {
			log.Printf("JWT parsing error: %s", err)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		issuer, err := url.Parse(claims.Issuer)
		if err != nil {
			log.Printf("valid issuer not found: %s", err)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if claims.Subject == "" {
			log.Printf("invalid CharmID found: %s", err)
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		c.Set("charm_id", claims.Subject)

		p := jwks.NewCachingProvider(issuer, 1*time.Hour)
		jwtValidator, err := validator.New(
			p.KeyFunc,
			validator.EdDSA,
			issuer.String(),
			[]string{"charm"},
		)
		if err != nil {
			log.Printf("could not create validator: %s", err)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		valid := false
		handler := func(http.ResponseWriter, *http.Request) { valid = true }
		middleware := jwtmiddleware.New(jwtValidator.ValidateToken)
		middleware.CheckJWT(http.HandlerFunc(handler)).ServeHTTP(c.Writer, c.Request)
		if valid {
			c.Next()
		} else {
			log.Printf("JWT validation failed")
			c.Abort()
		}
	}
}

func getClaims(auth string) (*jwt.RegisteredClaims, error) {
	tMinLen := len("Bearer ")
	if len(auth) <= tMinLen {
		return nil, fmt.Errorf("invalid header token")
	}

	encodedToken := auth[tMinLen:]
	p := jwt.Parser{}
	t, _, err := p.ParseUnverified(encodedToken, &jwt.RegisteredClaims{})
	if err != nil {
		return nil, err
	}

	return t.Claims.(*jwt.RegisteredClaims), nil
}
