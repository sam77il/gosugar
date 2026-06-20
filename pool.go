package sugar

import "sync"

var contextPool = sync.Pool{
	New: func() any {
		return &Context{
			Request:  &Request{},
			Response: &Response{},
		}
	},
}

func acquireContext() *Context {
	return contextPool.Get().(*Context)
}

func releaseContext(ctx *Context) {
	*ctx.Request = Request{}
	*ctx.Response = Response{}
	ctx.Header = nil
	ctx.Cookies = Cookies{}
	contextPool.Put(ctx)
}
