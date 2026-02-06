package handlers

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"retaildb-service/internal/auth"
	"retaildb-service/internal/llm"
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

	// LLM-powered natural language database query (read-only)
	r.POST("/query", func(c *gin.Context) {
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "LLM query is not configured"})
			return
		}
		model := os.Getenv("GEMINI_MODEL") // optional, defaults to gemini-2.0-flash-lite
		var body struct {
			Question string `json:"question"`
		}
		if err := c.BindJSON(&body); err != nil || strings.TrimSpace(body.Question) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "request body must contain a non-empty 'question' string"})
			return
		}
		generatedSQL, results, err := llm.Query(c.Request.Context(), db, apiKey, model, body.Question)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
				"sql":   generatedSQL,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"sql":     generatedSQL,
			"results": results,
		})
	})

	r.GET("/products", func(c *gin.Context) {
		limit := 20
		offset := 0
		if l := c.Query("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil {
				limit = v
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
				return
			}
		}
		if o := c.Query("offset"); o != "" {
			if v, err := strconv.Atoi(o); err == nil {
				offset = v
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
				return
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
	authGroup.Use(auth.Middleware())
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	// Serve a small interactive dashboard at the root for quick testing and demos
	r.GET("/", func(c *gin.Context) {
		html := `<!doctype html>
		<html>
		  <head>
		    <meta charset="utf-8" />
		    <meta name="viewport" content="width=device-width, initial-scale=1" />
		    <title>RetailDB Interactive Dashboard</title>
		    <style>
		      body { font-family: Arial, sans-serif; max-width: 900px; margin: 20px auto; }
		      h1 { margin-bottom: 0.2em }
		      .card { border: 1px solid #ddd; padding: 12px; margin: 12px 0; border-radius: 6px }
		      input, textarea, select { width: 100%; padding: 8px; margin-top: 6px; box-sizing: border-box }
		      button { padding: 8px 12px; margin-top: 8px }
		      pre { background: #f8f8f8; padding: 8px; overflow: auto }
		    </style>
		  </head>
		  <body>
		    <h1>RetailDB Interactive Dashboard</h1>
		    <p>Use these controls to interact with the running service. Tokens are stored in <code>localStorage</code>.</p>

		    <div class="card">
		      <h2>Login (get token)</h2>
		      <label>Username
		        <input id="login-user" value="admin" />
		      </label>
		      <label>Password
		        <input id="login-pass" value="password" type="password" />
		      </label>
		      <button onclick="login()">Login</button>
      <div>
        Token: <code id="token" style="white-space:pre-wrap; word-break:break-all">(none)</code>
        <button onclick="copyToken()">Copy</button>
        <button onclick="clearToken()">Clear</button>
        <a href="/docs" target="_blank" style="margin-left:12px">API Docs</a>
      </div>
		    </div>

		    <div class="card">
		      <h2>Public: List Products</h2>
		      <label>Limit<input id="limit" value="10" /></label>
		      <label>Offset<input id="offset" value="0" /></label>
		      <button onclick="listProducts()">List</button>
		      <pre id="list-result"></pre>
		    </div>

		    <div class="card">
		      <h2>Public: Get Product by ID</h2>
		      <label>Product ID<input id="get-id" /></label>
		      <button onclick="getProduct()">Get</button>
		      <pre id="get-result"></pre>
		    </div>

		    <div class="card">
		      <h2>Admin: Create Product (requires Bearer token)</h2>
		      <label>Product ID<input id="create-id" /></label>
		      <label>Product Name<input id="create-name" /></label>
		      <button onclick="createProduct()">Create</button>
		      <pre id="create-result"></pre>
		    </div>

		    <div class="card">
		      <h2>Admin: Update Modified Fields (JSON)</h2>
		      <label>Product ID<input id="update-id" /></label>
		      <label>JSON body<textarea id="update-body" rows="4">{"modified_product_name":"New name"}</textarea></label>
		      <button onclick="updateProduct()">Update</button>
		      <pre id="update-result"></pre>
		    </div>

		    <div class="card">
		      <h2>Admin: Delete Product</h2>
		      <label>Product ID<input id="delete-id" /></label>
		      <button onclick="deleteProduct()">Delete</button>
		      <pre id="delete-result"></pre>
		    </div>

		    <div class="card">
		      <h2>AI Query (Natural Language &rarr; SQL)</h2>
		      <p style="margin:0 0 6px;color:#666">Ask a question about the database in plain English. Uses Gemini to generate a read-only SQL query.</p>
		      <label>Question<input id="query-question" placeholder="e.g. What are the top 5 most expensive products?" /></label>
		      <button onclick="aiQuery()">Ask</button>
		      <details style="margin-top:6px"><summary>Generated SQL</summary><pre id="query-sql"></pre></details>
		      <pre id="query-result"></pre>
		    </div>

		    <script>
      function setToken(t) {
        localStorage.setItem('rdb_token', t);
        const el = document.getElementById('token');
        if (el) el.textContent = t || '(none)';
      }
      function getToken() { return localStorage.getItem('rdb_token') }
      function copyToken() {
        const t = getToken();
        if (!t) { showNotif('No token to copy'); return; }
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(t).then(() => showNotif('Token copied'), () => showNotif('Copy failed'))
        } else {
          // fallback
          const ta = document.createElement('textarea'); ta.value = t; document.body.appendChild(ta); ta.select(); try { document.execCommand('copy'); showNotif('Token copied') } catch (e) { showNotif('Copy failed') } finally { document.body.removeChild(ta) }
        }
      }
      function clearToken() { setToken(''); showNotif('Token cleared') }
      function showNotif(msg) { console.log('notif:', msg); alert(msg) }
      (function(){ setToken(getToken()); })();

      async function login() {
        const u = document.getElementById('login-user').value;
        const p = document.getElementById('login-pass').value;
        const res = await fetch('/auth/login', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({username:u, password:p})});
        let json = {}
        try { json = await res.json() } catch (e) { json = {status: res.status} }
        if (res.ok && json.token) { setToken(json.token) } else { alert('login failed: '+JSON.stringify(json)) }
      }

		      async function listProducts(){
        try {
          const l = document.getElementById('limit').value || 10;
          const o = document.getElementById('offset').value || 0;
          const res = await fetch('/products?limit=' + encodeURIComponent(l) + '&offset=' + encodeURIComponent(o));
          const body = await res.json();
          document.getElementById('list-result').innerText = JSON.stringify(body, null, 2);
        } catch (e) {
          document.getElementById('list-result').innerText = 'error: ' + e;
        }
      }

      async function getProduct(){
        try {
          const id = document.getElementById('get-id').value;
          const res = await fetch('/products/' + encodeURIComponent(id));
          const body = await res.json();
          document.getElementById('get-result').innerText = JSON.stringify(body, null, 2);
        } catch (e) {
          document.getElementById('get-result').innerText = 'error: ' + e;
        }
      }

      async function createProduct(){
        try {
          const token = getToken();
          const id = document.getElementById('create-id').value;
          const name = document.getElementById('create-name').value;
          const headers = {'Content-Type':'application/json'};
          if (token) headers['Authorization'] = 'Bearer ' + token;
          const res = await fetch('/admin/products', {method:'POST', headers: headers, body: JSON.stringify({product_id:id, product_name:name})});
          let out; try { out = await res.json() } catch(e) { out = {status:res.status} }
          document.getElementById('create-result').innerText = JSON.stringify(out, null, 2);
        } catch (e) {
          document.getElementById('create-result').innerText = 'error: ' + e;
        }
      }

      async function updateProduct(){
		        const token = getToken();
		        const id = document.getElementById('update-id').value;
		        let body = {};
		        try { body = JSON.parse(document.getElementById('update-body').value) } catch(e) { alert('invalid JSON'); return }
			const headers = {'Content-Type':'application/json'};
			if (token) headers['Authorization'] = 'Bearer ' + token;
			const res = await fetch('/admin/products/' + encodeURIComponent(id), {
				method:'PUT',
				headers: headers,
				body: JSON.stringify(body)
			});
			let out; try { out = await res.json() } catch(e) { out = {status:res.status} }
			document.getElementById('update-result').innerText = JSON.stringify(out, null, 2);
		}

		async function deleteProduct(){
			const token = getToken();
			const id = document.getElementById('delete-id').value;
			const headers = {};
			if (token) headers['Authorization'] = 'Bearer ' + token;
			const res = await fetch('/admin/products/' + encodeURIComponent(id), {method:'DELETE', headers: headers});
			let out; try { out = await res.json() } catch(e) { out = {status:res.status} }
			document.getElementById('delete-result').innerText = JSON.stringify(out, null, 2);
		}

		async function aiQuery(){
			try {
				const q = document.getElementById('query-question').value;
				if (!q) { alert('Enter a question'); return; }
				document.getElementById('query-result').innerText = 'Thinking...';
				document.getElementById('query-sql').innerText = '';
				const res = await fetch('/query', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({question:q})});
				const body = await res.json();
				document.getElementById('query-sql').innerText = body.sql || '(none)';
				if (body.results) {
					document.getElementById('query-result').innerText = JSON.stringify(body.results, null, 2);
				} else {
					document.getElementById('query-result').innerText = JSON.stringify(body, null, 2);
				}
			} catch(e) {
				document.getElementById('query-result').innerText = 'error: ' + e;
			}
		}
		    </script>
		  </body>
		</html>`
		c.Data(200, "text/html; charset=utf-8", []byte(html))
	})

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
