package sugar

import "strings"

type treeNode struct {
	literal      string
	paramName    string
	wildcardName string
	isParam      bool
	isWildcard   bool
	children     []*treeNode
	route        *route
}

func (n *treeNode) childFor(seg string) *treeNode {
	isParam := len(seg) > 0 && seg[0] == ':'
	isWild := len(seg) > 0 && seg[0] == '*'
	for _, c := range n.children {
		switch {
		case isParam && c.isParam:
			return c
		case isWild && c.isWildcard:
			return c
		case !isParam && !isWild && c.literal == seg:
			return c
		}
	}
	return nil
}

func (n *treeNode) insert(segments []string, r *route) {
	cur := n
	for _, seg := range segments {
		child := cur.childFor(seg)
		if child == nil {
			child = &treeNode{}
			switch {
			case len(seg) > 0 && seg[0] == ':':
				child.isParam = true
				child.paramName = seg[1:]
			case len(seg) > 0 && seg[0] == '*':
				child.isWildcard = true
				child.wildcardName = seg[1:]
			default:
				child.literal = seg
			}
			cur.children = append(cur.children, child)
		}
		cur = child
	}
	cur.route = r
}

func (n *treeNode) match(segments []string, params map[string]string) (*route, bool) {
	if len(segments) == 0 {
		return n.route, n.route != nil
	}
	seg := segments[0]

	for _, c := range n.children {
		if !c.isParam && !c.isWildcard && c.literal == seg {
			if r, ok := c.match(segments[1:], params); ok {
				return r, true
			}
		}
	}
	for _, c := range n.children {
		if c.isParam {
			params[c.paramName] = seg
			if r, ok := c.match(segments[1:], params); ok {
				return r, true
			}
			delete(params, c.paramName)
		}
	}
	for _, c := range n.children {
		if c.isWildcard {
			params[c.wildcardName] = strings.Join(segments, "/")
			return c.route, c.route != nil
		}
	}
	return nil, false
}

type routeTree struct {
	roots map[string]*treeNode
}

func buildRouteTree(routers []*Router) *routeTree {
	t := &routeTree{roots: make(map[string]*treeNode)}
	for _, router := range routers {
		for _, r := range router.routes {
			root := t.roots[r.method]
			if root == nil {
				root = &treeNode{}
				t.roots[r.method] = root
			}
			root.insert(r.segments, r)
		}
	}
	return t
}

func (t *routeTree) match(method string, segments []string) (*route, map[string]string, bool) {
	root := t.roots[method]
	if root == nil {
		return nil, nil, false
	}
	params := make(map[string]string)
	r, ok := root.match(segments, params)
	if !ok {
		return nil, nil, false
	}
	return r, params, true
}
