package main

import (
	"fmt"
	"gosugar/sugar"
)

func main() {
	server := sugar.New(sugar.Config{
		Port: 8080,
	})
	// creating router
	router := server.Router()

	router.Static("./static", "/files")

	server.Middleware("/ipa/*", func(ctx *sugar.SugarContext) error {
		fmt.Println(ctx.Request.Method)
		ctx.Header.Add("asa", "mitaka")
		ctx.Header.Add("lol", "yes")
		ctx.Header.Add("goofy", "yes")
		return ctx.Response.Status(300).Redirect("/api")
	})

	ipa := server.Group("/ipa")
	ipa.Get("/users", func (ctx *sugar.SugarContext) error {
		return ctx.Response.Status(200).JSON(sugar.J{"success": true})
	})

	router.Get("/api", func (ctx *sugar.SugarContext) error {
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