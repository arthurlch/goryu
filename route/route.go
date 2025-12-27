package route

// Route holds all the information for a registered route.
// This is extracted to avoid circular imports between router and context packages.
type Route struct {
	Method      string
	Path        string
	Handler     interface{} // Will hold context.HandlerFunc to avoid circular imports
	Name        string
	Description string      // Route description for documentation
	ParamNames  []string    // Optimization: Store param names to avoid map usage in traversal
	
	// Router field will be set by the router package using SetRouter method
	Router interface{} // Will hold *router.Router to avoid circular imports
}

func (r *Route) SetRouter(router interface{}) {
	r.Router = router
}

func (r *Route) GetRouter() interface{} {
	return r.Router
}

func (r *Route) SetName(name string) *Route {
	r.Name = name
	// If the router has a SetRouteName method, use it
	if r.Router != nil {
		// Call the router's SetRouteName method if it has one
		// This will be implemented by the router package
		if routerWithSetName, ok := r.Router.(interface{ SetRouteName(*Route, string) *Route }); ok {
			return routerWithSetName.SetRouteName(r, name)
		}
	}
	return r
}

func (r *Route) GetHandler() interface{} {
	return r.Handler
}

func (r *Route) GetName() string   { return r.Name }
func (r *Route) GetPath() string   { return r.Path }
func (r *Route) GetMethod() string { return r.Method }