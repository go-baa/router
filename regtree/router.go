package regtree

import (
	"fmt"
	"sync"

	"github.com/go-baa/baa"
)

// Router provlider router for baa
type Router struct {
	autoHead          bool
	autoTrailingSlash bool
	groups            []*group
	nodes             [baa.RouteLength]*Tree
	nameNodes         map[string]*Node
	mu                sync.RWMutex
	baa               *baa.Baa
}

// Node is a router node
type Node struct {
	paramNum int
	format   string
	name     string
	root     *Router
}

// group route
type group struct {
	pattern  string
	handlers []baa.HandlerFunc
}

// New create a router instance
func New(b *baa.Baa) baa.Router {
	r := new(Router)
	for _, i := range baa.RouterMethods {
		r.nodes[i] = NewTree("/", nil)
	}
	r.nameNodes = make(map[string]*Node)
	r.groups = make([]*group, 0)
	r.baa = b
	return r
}

// newNode create a route node
func newNode(format string, paramsNum int, router *Router) *Node {
	n := new(Node)
	n.format = format
	n.paramNum = paramsNum
	n.root = router
	return n
}

// newGroup create a group router
func newGroup() *group {
	g := new(group)
	g.handlers = make([]baa.HandlerFunc, 0)
	return g
}

// SetAutoHead sets the value who determines whether add HEAD method automatically
// when GET method is added. Combo router will not be affected by this value.
func (r *Router) SetAutoHead(v bool) {
	r.autoHead = v
}

// SetAutoTrailingSlash optional trailing slash.
func (r *Router) SetAutoTrailingSlash(v bool) {
	r.autoTrailingSlash = v
}

// Match find matched route then returns handlers and name
func (r *Router) Match(method, uri string, c *baa.Context) ([]baa.HandlerFunc, string) {
	return r.nodes[baa.RouterMethods[method]].Get(uri, c)
}

// URLFor use named route return format url
func (r *Router) URLFor(name string, args ...interface{}) string {
	if name == "" {
		return ""
	}
	node := r.nameNodes[name]
	if node == nil || len(node.format) == 0 {
		return ""
	}
	format := make([]byte, len(node.format))
	copy(format, node.format)
	for i := node.paramNum + 1; i <= len(args); i++ {
		format = append(format, "%v"...)
	}
	return fmt.Sprintf(string(format), args...)
}

// Routes returns registered route uri in a string slice
func (r *Router) Routes() map[string][]string {
	routes := make(map[string][]string)
	for _, method := range baa.RouterMethodName {
		routes[method] = make([]string, 0)
	}
	for k := range r.nodes {
		routes[baa.RouterMethodName[k]] = r.routes(r.nodes[k])
	}

	return routes
}

// routes print the route table
func (r *Router) routes(l *Tree) []string {
	if l == nil {
		return nil
	}
	var data []string
	if l.handlers != nil {
		data = append(data, l.String())
	}
	l.schildren = append(l.schildren, l.rchildren...)
	for i := range l.schildren {
		if l.schildren[i] != nil {
			data = append(data, r.routes(l.schildren[i])...)
		}
	}

	return data
}

// NamedRoutes returns named route uri in a string slice
func (r *Router) NamedRoutes() map[string]string {
	routes := make(map[string]string)
	for k, v := range r.nameNodes {
		routes[k] = v.format
	}
	return routes
}

// Add registers a new handle with the given method, pattern and handlers.
// add check training slash option.
func (r *Router) Add(method, pattern string, handlers []baa.HandlerFunc) baa.RouteNode {
	if method == "GET" && r.autoHead {
		r.add("HEAD", pattern, handlers)
	}
	if r.autoTrailingSlash && (len(pattern) > 1 || len(r.groups) > 0) {
		var index byte
		if len(pattern) > 0 {
			index = pattern[len(pattern)-1]
		}
		if index == '/' {
			r.add(method, pattern[:len(pattern)-1], handlers)
		} else {
			r.add(method, pattern+"/", handlers)
		}
	}
	return r.add(method, pattern, handlers)
}

// GroupAdd add a group route has same prefix and handle chain
func (r *Router) GroupAdd(pattern string, f func(), handlers []baa.HandlerFunc) {
	g := newGroup()
	g.pattern = pattern
	g.handlers = handlers
	r.groups = append(r.groups, g)

	f()

	r.groups = r.groups[:len(r.groups)-1]
}

// add registers a new request handle with the given method, pattern and handlers.
func (r *Router) add(method, pattern string, handlers []baa.HandlerFunc) *Node {
	if _, ok := baa.RouterMethods[method]; !ok {
		panic("Router.add: unsupport http method [" + method + "]")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// check group set
	if len(r.groups) > 0 {
		var gpattern string
		var ghandlers []baa.HandlerFunc
		for i := range r.groups {
			gpattern += r.groups[i].pattern
			if len(r.groups[i].handlers) > 0 {
				ghandlers = append(ghandlers, r.groups[i].handlers...)
			}
		}
		pattern = gpattern + pattern
		ghandlers = append(ghandlers, handlers...)
		handlers = ghandlers
	}

	// check pattern (for training slash move behind group check)
	if pattern == "" {
		panic("Router.add: pattern can not be emtpy!")
	}
	if pattern[0] != '/' {
		panic("Router.add: pattern must begin /")
	}

	nameNode := newNode("", 0, r)

	for i := 0; i < len(handlers); i++ {
		handlers[i] = baa.WrapHandlerFunc(handlers[i])
	}

	node := r.nodes[baa.RouterMethods[method]].Add(pattern, handlers, nameNode)
	if node == nil {
		panic("Router.add: tree.add error")
	}

	nameNode.format = string(node.formatStr())
	nameNode.paramNum = len(node.params)
	return nameNode
}

// Name set name of route
func (n *Node) Name(name string) {
	if name == "" {
		return
	}
	n.name = name
	n.root.nameNodes[name] = n
}
