package docs

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const referenceHTML = `<!doctype html>
<html>
  <head>
    <title>COLABORA API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <div id="app"></div>
    <script
      src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.66.1"
      integrity="sha384-Ew/tMn10mwKgtOrbm80/2triFQ8aLGRERlQr828oZ21Sk8xxHRBl+dccrP1fj+CS"
      crossorigin="anonymous"
    ></script>
    <script>
      Scalar.createApiReference('#app', {
        hideModels: true,
        url: '/docs/openapi.yaml',
      })
    </script>
  </body>
</html>`

// RegisterRoutes serves one OpenAPI document per module (each self-contained,
// not sharing $ref'd components across files), a combined document merging
// all of them at /docs/openapi.yaml (see buildMergedOpenAPI), and a single
// Scalar-powered API reference page pointed at that combined document so
// every module shows up as its own sidebar section in one flat list instead
// of a dropdown you have to switch between.
// See https://scalar.com/products/api-references/getting-started and
// https://scalar.com/products/api-references/configuration.
//
// `hideModels: true` drops the "Models"/schemas sidebar section — Scalar still
// resolves $ref internally to render each endpoint's request/response bodies,
// this only hides the separate schema-browsing UI.
//
// Adding a new module's docs: add its own `docs/<module>.yaml` (self-contained
// — copy the small ApiResponse/BadRequest/Unauthorized boilerplate from an
// existing file rather than trying to share it across documents) plus one
// `server.StaticFile` line below and one entry in `moduleDocFiles`
// (openapi_merge.go).
//
// The CDN script is pinned to a specific @scalar/api-reference version with a
// matching SRI hash instead of the unversioned "latest" tag, so a compromised
// CDN can't silently swap the script. Bumping the Scalar version means
// updating both the version in the src URL and the integrity hash together
// (recompute via: curl -sL <url> | openssl dgst -sha384 -binary | openssl base64 -A).
func RegisterRoutes(server *gin.Engine) {
	server.StaticFile("/docs/health.yaml", "./docs/health.yaml")
	server.StaticFile("/docs/auth.yaml", "./docs/auth.yaml")
	server.StaticFile("/docs/user.yaml", "./docs/user.yaml")

	server.GET("/docs/openapi.yaml", func(ctx *gin.Context) {
		body, err := buildMergedOpenAPI()
		if err != nil {
			ctx.Data(http.StatusInternalServerError, "text/plain; charset=utf-8", []byte(err.Error()))
			return
		}
		ctx.Data(http.StatusOK, "application/yaml; charset=utf-8", body)
	})

	server.GET("/docs", func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(referenceHTML))
	})
}
