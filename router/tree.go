package router

import (
	"strings"

	"github.com/arthurlch/goryu/context"
)

// SUing Radix Tree
type node struct {
	path     string
	part     string
	children []*node
	isWild   bool
	handler  context.HandlerFunc
	route    *Route
}

func (n *node) insert(path string, parts []string, height int, route *Route) {
	if len(parts) == height {
		n.path = path
		n.handler = route.Handler
		n.route = route
		return
	}

	part := parts[height]
	child := n.matchChild(part)
	if child == nil {
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}
	child.insert(path, parts, height+1, route)
}

func (n *node) find(parts []string, height int) (*node, map[string]string) {
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.handler == nil {
			return nil, nil
		}
		params := make(map[string]string)
		if strings.HasPrefix(n.part, "*") {
			params[n.part[1:]] = strings.Join(parts[height:], "/")
		}
		return n, params
	}

	part := parts[height]
	params := make(map[string]string)

	for _, child := range n.children {
		if child.part == part || child.isWild {
			if child.isWild {
				params[child.part[1:]] = part
			}
			result, foundParams := child.find(parts, height+1)
			if result != nil {
				for k, v := range foundParams {
					params[k] = v
				}
				return result, params
			}
		}
	}

	return nil, nil
}

func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}
