package main

import (
	"fmt"
	"gosugar/sugar"
)

func main() {
	server := sugar.New(sugar.Config{
		Port: 8080,
	})

	server.Get("/api", func (ctx *sugar.SugarContext) {
		if ctx.Request.Header.Get("lol") == "yes" {
			ctx.Request.Next()
			return
		}
		ctx.Response.Status(400).JSON(map[string]any{"success": false, "message": "wrong method"})
	}, func (ctx *sugar.SugarContext)  {
		fmt.Println("yes lol")

		if ctx.Request.Header.Get("goofy") == "yes" {
			ctx.Request.Next()
			return
		}
		ctx.Response.Status(400).JSON(map[string]any{"success": false})
	}, func (ctx *sugar.SugarContext) {
		ctx.Response.Status(200).JSON(map[string]any{"success": true})
	})

	server.Listen()
}