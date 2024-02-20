package main

import (
	"context"
	"net/http"

	"github.com/HannesOberreiter/gbif-extinct/components"
	"github.com/HannesOberreiter/gbif-extinct/internal"
	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {

	e := echo.New()

	/* Middleware */
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	/* Routes */
	e.GET("/", index)
	e.GET("/table", table)
	e.File("/favicon.ico", "./assets/favicon.png")
	e.Static("/assets", "./assets")

	e.Logger.Fatal(e.Start(":1323"))
}

func index(c echo.Context) error {
	var payload internal.Payload
	err := c.Bind(&payload)
	if err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}
	table := internal.GetTableData(payload)
	return render(c,
		http.StatusAccepted,
		components.Page(table, payload))

}

func table(c echo.Context) error {
	var payload internal.Payload
	err := c.Bind(&payload)
	if err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}
	table := internal.GetTableData(payload)

	if c.Request().Header.Get("HX-Request") == "" {
		return c.JSON(http.StatusOK, table)
	}

	querystring := c.QueryString()
	c.Response().Header().Set("HX-Push-Url", "/?"+querystring)

	return render(c,
		http.StatusAccepted,
		components.Table(table, payload))
}

func render(c echo.Context, status int, t templ.Component) error {
	c.Response().Writer.WriteHeader(status)
	err := t.Render(context.Background(), c.Response().Writer)
	if err != nil {
		return c.String(http.StatusInternalServerError, "failed to render response template")
	}
	return nil
}
