package main

import (
	"fmt"
	"gosugar/sugar"
	"net/http"
)

func main() {
	server := sugar.New(sugar.Config{
		Port: 8080,
		Static: "./static",
	})

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/files", fs)

	server.Get("/api", func (ctx *sugar.SugarContext) error {
		fmt.Println(ctx.Request.IP)
		if ctx.Request.Header.Get("lol") == "yes" {
			ctx.Request.Next()
			return nil
		}
		return ctx.Response.Status(400).JSON(map[string]any{"success": false, "message": "wrong method"})
	}, func (ctx *sugar.SugarContext) error  {
		fmt.Println("yes lol")

		if ctx.Request.Header.Get("goofy") == "yes" {
			ctx.Request.Next()
			return nil
		}
		return ctx.Response.Status(400).JSON(sugar.J{"success": false})
	}, func (ctx *sugar.SugarContext) error {
		return ctx.Response.Status(200).JSON(map[string]any{"success": true})
	})

	server.Listen()
}