package router

import (
	"fmt"

	"github.com/arthurlch/goryu/goryuctx"
)

type nodeType uint8

const (
	static nodeType = iota // default
	root
	param
	catchAll
)

// node represents a node in the Radix tree
// I wasn't aware about this data structure before, but it's quite efficient for routing and it seems to be used in other frameworks too.
// MEMO: https://en.wikipedia.org/wiki/Radix_tree
type node struct {
	path      string
	indices   string
	wildChild bool
	children  []*node

	nType    nodeType
	priority uint32
	handler  goryuctx.HandlerFunc
	route    *Route
}

// Increment priority of the given child and reorder if necessary
func (n *node) incrementChildPrio(pos int) int {
	cs := n.children
	cs[pos].priority++
	prio := cs[pos].priority

	// Verify priority ordering (bubble up)
	newPos := pos
	for newPos > 0 && cs[newPos-1].priority < prio {
		// Swap
		cs[newPos-1], cs[newPos] = cs[newPos], cs[newPos-1]
		newPos--
	}

	// Update indices string to match new order
	if newPos != pos {
		n.indices = n.indices[:newPos] + // prefix
			n.indices[pos:pos+1] + // moved char
			n.indices[newPos:pos] + // middle
			n.indices[pos+1:] // suffix
	}

	return newPos
}

// addRoute adds a node with the given handle to the path.
// Not thread-safe.
func (n *node) addRoute(path string, handler goryuctx.HandlerFunc, route *Route) error {
	fullPath := path
	n.priority++

	// Empty tree
	if len(n.path) == 0 && len(n.children) == 0 {
		return n.insertChild(path, fullPath, handler, route)
	}

walk:
	for {
		// Find longest common prefix
		i := longestCommonPrefix(path, n.path)

		// Split edge
		if i < len(n.path) {
			child := node{
				path:      n.path[i:],
				indices:   n.indices,
				wildChild: n.wildChild,
				children:  n.children,
				handler:   n.handler,
				route:     n.route,
				priority:  n.priority - 1,
				nType:     static,
			}

			// Update node type for params/catchAll children
			for _, c := range child.children {
				if c.nType == param || c.nType == catchAll {
					child.nType = static
					break
				}
			}

			n.children = []*node{&child}
			// []byte for proper unicode handling
			n.indices = string([]byte{n.path[i]})
			n.path = path[:i]
			n.handler = nil
			n.route = nil
			n.wildChild = false
		}

		// Make new node a child of this node
		if i < len(path) {
			path = path[i:]

			c := path[0]

			// Check if we're at a wildcard parent node
			if n.wildChild && (n.nType == static || n.nType == root) {
				// Continue with wildcard child
				n = n.children[len(n.children)-1]
				n.priority++

				// Update parent node's max priority
				if n.priority > n.children[0].priority {
					n.children[0] = n
				}

				continue walk
			}

			// Check if a child with the next path byte exists
			for i, max := 0, len(n.indices); i < max; i++ {
				if c == n.indices[i] {
					i = n.incrementChildPrio(i)
					n = n.children[i]
					continue walk
				}
			}

			// Otherwise insert new node
			if c != ':' && c != '*' {
				// []byte for proper unicode handling
				n.indices += string([]byte{c})
				child := &node{}
				n.children = append(n.children, child)
				n.incrementChildPrio(len(n.indices) - 1)
				n = child
			}
			return n.insertChild(path, fullPath, handler, route)
		}

		// Otherwise add handle to current node
		if n.handler != nil {
			return fmt.Errorf("route '%s' already exists", fullPath)
		}
		n.handler = handler
		n.route = route
		return nil
	}
}

func (n *node) insertChild(path, fullPath string, handler goryuctx.HandlerFunc, route *Route) error {
	for {
		// Find prefix until first wildcard
		wildcard, i, valid := findWildcard(path)
		if i < 0 { // No wildcard
			break
		}

		// The wildcard name must not contain ':' and '*'
		if !valid {
			return fmt.Errorf("only one wildcard per path segment is allowed, has: '%s' in path '%s'", wildcard, fullPath)
		}

		// Check basic validity of catch-all placement before modifying path
		if wildcard[0] == '*' && i > 0 && path[i-1] != '/' {
			return fmt.Errorf("no / before catch-all in path '%s'", fullPath)
		}

		// Check if the wildcard has an existing node
		if i > 0 {
			n.path = path[:i]
			path = path[i:]
		}

		if wildcard[0] == ':' { // param
			// Check if this node already has a wildcard child
			if n.wildChild {
				n = n.children[len(n.children)-1]
				n.priority++

				// Check if the wildcard matches
				if n.path != wildcard {
					return fmt.Errorf("wildcard route '%s' conflicts with existing wildcard route '%s'", wildcard, n.path)
				}

				if len(wildcard) < len(path) {
					path = path[len(wildcard):]

					// Add static child if doesn't exist
					c := path[0]
					for i, max := 0, len(n.indices); i < max; i++ {
						if c == n.indices[i] {
							n = n.children[i]
							continue
						}
					}

					// Need to create a new static child
					n.indices += string([]byte{c})
					child := &node{}
					n.children = append(n.children, child)
					n = child
					continue
				}
			} else {
				// Insert wildcard node
				n.wildChild = true
				child := &node{
					nType:    param,
					path:     wildcard,
					priority: 1,
				}
				n.children = append(n.children, child)
				n = child

				// If the path doesn't end with the wildcard
				if len(wildcard) < len(path) {
					path = path[len(wildcard):]

					child := &node{
						priority: 1,
					}
					n.children = []*node{child}
					n = child
					continue
				}
			}

			n.handler = handler
			n.route = route
			return nil
		}

		// catchAll
		// The catch-all should consume the rest of the path
		if len(wildcard) < len(path) {
			return fmt.Errorf("catch-all routes are only allowed at the end of the path in path '%s'", fullPath)
		}

		// Removed the check for trailing slash since we handle paths differently in full radix tree

		// Check if this node already has a wildcard child
		if n.wildChild {
			n = n.children[len(n.children)-1]

			// Check for conflicts
			if n.path != wildcard {
				return fmt.Errorf("catch-all route '%s' conflicts with existing route '%s'", wildcard, n.path)
			}
			if n.handler != nil {
				return fmt.Errorf("route '%s' already exists", fullPath)
			}
		} else {
			// Insert wildcard node
			n.wildChild = true
			child := &node{
				nType:    catchAll,
				path:     wildcard,
				priority: 1,
			}
			n.children = append(n.children, child)
			n = child
		}

		n.handler = handler
		n.route = route
		return nil
	}

	// If no wildcard was found, simply insert the path and handle
	n.path = path
	n.handler = handler
	n.route = route
	return nil
}

// Returns the value associated with the given pathas
func (n *node) getValue(path string, params []string, unescape bool) (goryuctx.HandlerFunc, *Route, []string, bool) {
walk: // Outer loop label
	for {
		prefix := n.path
		if n.nType == param {
			end := 0
			for end < len(path) && path[end] != '/' {
				end++
			}

			// Save param value
			if cap(params) < len(params)+1 {
				newParams := make([]string, len(params), len(params)+4)
				copy(newParams, params)
				params = newParams
			}
			params = append(params, path[:end])

			// Continue deeper
			if end < len(path) {
				if len(n.children) > 0 {
					path = path[end:]
					n = n.children[0]
					continue walk
				}

				// No deeper path found
				return nil, nil, params, false
			}

			if n.handler != nil {
				return n.handler, n.route, params, false
			}
			// Check for trailing slash recommendation
			// should omit nil check; len() for nil slices is defined as zero
			// Well f..u linter chan
			return nil, nil, params, n.children != nil && len(n.children) == 1 && n.children[0].path == "/"

		} else if n.nType == catchAll {
			params = append(params, path)
			return n.handler, n.route, params, false
		}

		if len(path) > len(prefix) {
			if path[:len(prefix)] == prefix {
				path = path[len(prefix):]

				// First, look for static match (highest priority)
				idxc := path[0]
				for i, c := range []byte(n.indices) {
					if c == idxc {
						n = n.children[i]
						continue walk
					}
				}

				// If no static match and this node has a wildcard child, check it
				if n.wildChild {
					n = n.children[len(n.children)-1]
					continue walk
				}

				// Nothing found. Check if a route with a trailing slash exists
				tsr := path == "/" && n.handler != nil
				return nil, nil, params, tsr
			}
		} else if path == prefix {
			// Exact match
			if n.handler != nil {
				return n.handler, n.route, params, false
			}

			// If this node has a wildcard child with empty path
			if n.wildChild && n.children[len(n.children)-1].nType == catchAll {
				n = n.children[len(n.children)-1]
				params = append(params, "")
				return n.handler, n.route, params, false
			}

			// Check for trailing slash recommendation
			// Check if path+"/" would have a handler
			for i, c := range []byte(n.indices) {
				if c == '/' {
					n = n.children[i]
					// Check if adding a slash would find a route
					tsr := (len(n.path) == 1 && n.handler != nil) ||
						(n.wildChild && n.children[len(n.children)-1].nType == catchAll)
					return nil, nil, params, tsr
				}
			}

			return nil, nil, params, false
		} else {
			// Check if we're one character away from a match (trailing slash case)
			// e.g., path="users" and prefix="users/"
			if len(prefix) == len(path)+1 && prefix[len(path)] == '/' && path == prefix[:len(path)] {
				// We need a trailing slash
				return nil, nil, params, true
			}
		}

		// Nothing found
		return nil, nil, params, false
	}
}

// walk recursively traverses the tree and calls the callback for each route found
func (n *node) walk(fn func(path string, route *Route)) {
	if n.handler != nil && n.route != nil {
		// Use stored route path
		fn(n.route.Path, n.route)
	}

	// Walk all children
	for _, child := range n.children {
		child.walk(fn)
	}
}

// Search for a wildcard segment and check the name for invalid characters.
// Returns -1 as index, if no wildcard was found.
func findWildcard(path string) (wildcard string, i int, valid bool) {
	// Find start
	for start, c := range []byte(path) {
		// A wildcard starts with ':' (param) or '*' (catch-all)
		if c != ':' && c != '*' {
			continue
		}

		// Find end and check for invalid characters
		valid = true
		for end, c := range []byte(path[start+1:]) {
			switch c {
			case '/':
				return path[start : start+1+end], start, valid
			case ':', '*':
				valid = false
			}
		}
		return path[start:], start, valid
	}
	return "", -1, false
}

func longestCommonPrefix(a, b string) int {
	i := 0
	max := min(len(a), len(b))
	for i < max && a[i] == b[i] {
		i++
	}
	return i
}
