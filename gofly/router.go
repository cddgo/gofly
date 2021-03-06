package gofly

import (
	"net/http"
	"strings"
)

type router struct {
	roots    map[string]*node
	handlers map[string]HandlerFunc
}

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

func parsePattern(pattern string) []string {
	split := strings.Split(pattern, "/")

	parts := make([]string, 0)

	for _, item := range split {
		if item != "" {
			parts = append(parts, item)
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

// addRoute defines the methods to add handler router container
func (r *router) addRoute(method, pattern string, handler HandlerFunc) {
	key := method + "-" + pattern // "GET-/docs/:lang/doc"

	// Trie树添加动态路由部分
	parts := parsePattern(pattern) // ["docs", ":lang", "doc"]
	_, ok := r.roots[method]
	if !ok {
		r.roots[method] = &node{}
	}

	// /docs/:lang/doc   ["docs", ":lang", "doc"]   0
	r.roots[method].insert(pattern, parts, 0)
	r.handlers[key] = handler
}

func (r *router) getRoute(method, path string) (*node, map[string]string) {
	// Trie树获取动态路由部分
	pathParts := parsePattern(path)   // 请求传入的path 如：/docs/cn/doc
	params := make(map[string]string) // 存储参数 如： :lang=cn

	root, ok := r.roots[method]

	if !ok {
		return nil, nil
	}

	n := root.search(pathParts, 0)
	if n != nil {
		searchParts := parsePattern(n.pattern)
		for index, part := range searchParts {
			if part[0] == ':' {
				params[part[1:]] = pathParts[index] // params["lang"] = cn
			}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(pathParts[index:], "/")
				break
			}
		}
		return n, params
	}
	return nil, nil
}

func (r *router) getRoutes(method string) []*node {
	n, ok := r.roots[method]
	if !ok {
		return nil
	}
	nodes := make([]*node, 0)
	n.travel(&nodes)
	return nodes
}

// handle the request with the router table
func (r *router) handler(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)

	if n != nil {
		c.Params = params
		key := c.Method + "-" + n.pattern
		c.handlers = append(c.handlers, r.handlers[key])
	} else {
		c.handlers = append(c.handlers, func(c *Context) {
			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
		})
	}
	c.Next() // index:-1 ----> 中间件 A -> B -> r.handlers[key] -> [B -> A]
	// 每个middleware 以 c.Next()为分界线。先执行分界线上面的 -> 走到handler并执行完 -> 执行中间件分界线下面的
}
