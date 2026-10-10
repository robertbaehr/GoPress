// wp_is_rest_endpoint / wp_is_serving_rest_request (P2.65) — subset of
// wp-includes/rest-api.php and wp-includes/functions.php.
package wp

// IsServingRESTRequest reports whether this is a standalone REST API request
// (wp_is_serving_rest_request / REST_REQUEST). True when Scope.IsREST is set
// (pretty /wp-json/, ?rest_route=, or rest.Server.Handle entry).
func IsServingRESTRequest(c *Context) bool {
	if c == nil || c.Scope == nil {
		return false
	}
	return c.Scope.IsREST
}

// IsRESTEndpoint reports whether a REST endpoint request is currently being
// handled (wp_is_rest_endpoint). Covers standalone REST requests and in-flight
// REST dispatch (Scope.RESTDispatching). Filterable via wp_is_rest_endpoint.
func IsRESTEndpoint(c *Context) bool {
	is := IsServingRESTRequest(c)
	if !is && c != nil && c.Scope != nil && c.Scope.IsRESTDispatching() {
		is = true
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_is_rest_endpoint", is); v != nil {
			if b, ok := v.(bool); ok {
				return b
			}
		}
	}
	return is
}
