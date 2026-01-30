package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"retaildb-service/internal/auth"
	"retaildb-service/internal/models"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB) {
	r.POST("/auth/login", func(c *gin.Context) {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		if !auth.ValidateCredentials(body.Username, body.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		tok, err := auth.GenerateJWT(body.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": tok})
	})

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	r.GET("/products", func(c *gin.Context) {
		limit := 20
		offset := 0
		if l := c.Query("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil {
				limit = v
			}
		}
		if o := c.Query("offset"); o != "" {
			if v, err := strconv.Atoi(o); err == nil {
				offset = v
			}
		}
		prods, err := models.GetProducts(db, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, prods)
	})

	r.GET("/products/:id", func(c *gin.Context) {
		id := c.Param("id")
		p, err := models.GetProductByID(db, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if p == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, p)
	})

	authGroup := r.Group("/admin")
	authGroup.Use(auth.AuthMiddleware())
	{
		authGroup.PUT("/products/:id", func(c *gin.Context) {
			id := c.Param("id")
			var updates map[string]interface{}
			if err := c.BindJSON(&updates); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
				return
			}
			if err := models.UpdateModifiedFields(db, id, updates); err != nil {
				if err == sql.ErrNoRows {
					c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
					return
				}
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "updated"})
		})

		// Admin: delete a product (protected)
		authGroup.DELETE("/products/:id", func(c *gin.Context) {
			id := c.Param("id")
			if id == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
				return
			}
			if err := models.DeleteProduct(db, id); err != nil {
				if err == sql.ErrNoRows {
					c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "deleted", "product_id": id})
		})
	}

	// Serve OpenAPI definition and a minimal Swagger UI
	r.StaticFile("/openapi.yaml", "./openapi.yaml")
	r.GET("/docs", func(c *gin.Context) {
		html := `<!doctype html>
		<html>
		  <head>
		    <meta charset="utf-8" />
		    <meta name="viewport" content="width=device-width, initial-scale=1" />
		    <title>RetailDB API Docs</title>
		    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@4.18.3/swagger-ui.css" />
		  </head>
		  <body>
		    <div id="swagger-ui"></div>
		    <script src="https://unpkg.com/swagger-ui-dist@4.18.3/swagger-ui-bundle.js"></script>
		    <script>
		      window.onload = function() {
		        const ui = SwaggerUIBundle({
		          url: '/openapi.yaml',
		          dom_id: '#swagger-ui',
		          presets: [SwaggerUIBundle.presets.apis],
		        })
		      }
		    </script>
		  </body>
		</html>`
		c.Data(200, "text/html; charset=utf-8", []byte(html))
	})

	// Admin: create a new product (protected)
	authGroup.POST("/products", func(c *gin.Context) {
		var req models.CreateProductRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		// basic validation
		req.ProductID = strings.TrimSpace(req.ProductID)
		if req.ProductID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "product_id is required"})
			return
		}
		if err := models.CreateProduct(db, req); err != nil {
			if strings.Contains(err.Error(), "exists") {
				c.JSON(http.StatusConflict, gin.H{"error": "product already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"status": "created", "product_id": req.ProductID})
	})
}
