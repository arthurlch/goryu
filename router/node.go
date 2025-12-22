package router

import (
	"fmt"
	"strings"
)

// node represents a node in the routing tree
type node struct {
	part      string  // The path segment this node represents
	children  []*node // Static child nodes
	paramChild *node  // Parameter child (:param)
	wildcardChild *node // Wildcard child (*param)
	route     *Route  // Route handler if this is an endpoint
	routeWithSlash *Route  // Route handler for path with trailing slash (StrictRouting)
	isParam   bool    // Is this a parameter node
	isWildcard bool   // Is this a wildcard node
}

// insert adds a route to the tree
func (n *node) insert(fullPath string, parts []string, partIndex int, route *Route, hasTrailingSlash bool, router *Router) {
	// If we've processed all parts, this is the endpoint
	if partIndex == len(parts) {
		// Store route based on whether it has trailing slash
		if hasTrailingSlash && router.Config.StrictRouting {
			if n.routeWithSlash != nil {
				router.handleRouterError("Add", fmt.Sprintf("route '%s' already exists", fullPath))
				return
			}
			n.routeWithSlash = route
		} else {
			if n.route != nil {
				router.handleRouterError("Add", fmt.Sprintf("route '%s' already exists", fullPath))
				return
			}
			n.route = route
		}
		return
	}

	part := parts[partIndex]
	isParam := strings.HasPrefix(part, ":")
	isWildcard := strings.HasPrefix(part, "*")

	// Wildcard must be at the end
	if isWildcard && partIndex != len(parts)-1 {
		router.handleRouterError("Add", "wildcard must be at the end of the path")
		return
	}

	// Handle parameter routes
	if isParam {
		if n.paramChild == nil {
			n.paramChild = &node{
				part:    part,
				isParam: true,
			}
		}
		n.paramChild.insert(fullPath, parts, partIndex+1, route, hasTrailingSlash, router)
		return
	}

	// Handle wildcard routes
	if isWildcard {
		if n.wildcardChild == nil {
			n.wildcardChild = &node{
				part:       part,
				isWildcard: true,
			}
		}
		n.wildcardChild.insert(fullPath, parts, partIndex+1, route, hasTrailingSlash, router)
		return
	}

	// Handle static routes
	for _, child := range n.children {
		if child.part == part {
			child.insert(fullPath, parts, partIndex+1, route, hasTrailingSlash, router)
			return
		}
	}

	// Create new static child
	child := &node{part: part}
	n.children = append(n.children, child)
	child.insert(fullPath, parts, partIndex+1, route, hasTrailingSlash, router)
}

// find searches for a route in the tree
func (n *node) find(parts []string, partIndex int, hasTrailingSlash bool, strictRouting bool) (*node, map[string]string, *Route) {
	params := make(map[string]string)

	// If we've processed all parts, return this node if it has a route
	if partIndex == len(parts) {
		// Check for route based on trailing slash preference
		if strictRouting {
			// In strict routing mode, only return the route that matches the trailing slash
			if hasTrailingSlash {
				if n.routeWithSlash != nil {
					return n, params, n.routeWithSlash
				}
			} else {
				if n.route != nil {
					return n, params, n.route
				}
			}
		} else {
			// In non-strict mode, prefer exact match but fall back to either
			if hasTrailingSlash && n.routeWithSlash != nil {
				return n, params, n.routeWithSlash
			} else if n.route != nil {
				return n, params, n.route
			} else if n.routeWithSlash != nil {
				return n, params, n.routeWithSlash
			}
		}
		
		// If no direct route found, try wildcard routes (wildcard can match empty remaining path)
		if n.wildcardChild != nil {
			paramName := n.wildcardChild.part[1:] // Remove the '*'
			// Wildcard captures empty remaining path
			params[paramName] = ""
			if foundNode, foundParams, foundRoute := n.wildcardChild.find([]string{}, 0, hasTrailingSlash, strictRouting); foundNode != nil && foundRoute != nil {
				// Merge parameters
				for k, v := range foundParams {
					params[k] = v
				}
				return foundNode, params, foundRoute
			}
		}
		
		return nil, nil, nil
	}

	part := parts[partIndex]

	// Priority 1: Try static routes first
	for _, child := range n.children {
		if child.part == part {
			if foundNode, foundParams, foundRoute := child.find(parts, partIndex+1, hasTrailingSlash, strictRouting); foundNode != nil && foundRoute != nil {
				// Merge parameters
				for k, v := range foundParams {
					params[k] = v
				}
				return foundNode, params, foundRoute
			}
		}
	}

	// Priority 2: Try parameter routes
	if n.paramChild != nil {
		paramName := n.paramChild.part[1:] // Remove the ':'
		params[paramName] = part
		if foundNode, foundParams, foundRoute := n.paramChild.find(parts, partIndex+1, hasTrailingSlash, strictRouting); foundNode != nil && foundRoute != nil {
			// Merge parameters
			for k, v := range foundParams {
				params[k] = v
			}
			return foundNode, params, foundRoute
		}
		// Clean up param if no match found
		delete(params, paramName)
	}

	// Priority 3: Try wildcard routes
	if n.wildcardChild != nil {
		paramName := n.wildcardChild.part[1:] // Remove the '*'
		// Wildcard captures the rest of the path
		remainingPath := strings.Join(parts[partIndex:], "/")
		params[paramName] = remainingPath
		if foundNode, foundParams, foundRoute := n.wildcardChild.find([]string{}, 0, hasTrailingSlash, strictRouting); foundNode != nil && foundRoute != nil {
			// Merge parameters
			for k, v := range foundParams {
				params[k] = v
			}
			return foundNode, params, foundRoute
		}
	}

	return nil, nil, nil
}

// parsePath splits a URL path into segments
func parsePath(path string) []string {
	if path == "/" {
		return []string{}
	}
	
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}
	
	return strings.Split(path, "/")
}

// walk recursively traverses the tree and calls the callback for each route found
func (n *node) walk(fn func(path string, route *Route)) {
	// If current node has a route, call callback
	if n.route != nil {
		fn(n.route.Path, n.route)
	}
	if n.routeWithSlash != nil {
		fn(n.routeWithSlash.Path, n.routeWithSlash)
	}

	// Traverse children
	for _, child := range n.children {
		child.walk(fn)
	}
	if n.paramChild != nil {
		n.paramChild.walk(fn)
	}
	if n.wildcardChild != nil {
		n.wildcardChild.walk(fn)
	}
}