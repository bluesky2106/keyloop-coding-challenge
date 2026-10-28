// Starts Swagger UI on the spec served next to this page. It is a separate file, not an
// inline script, because the Content-Security-Policy does not allow inline scripts.
window.ui = SwaggerUIBundle({
  url: 'openapi.yaml',
  dom_id: '#swagger-ui',
  deepLinking: true,
})
