package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HannesOberreiter/gbif-extinct/components"
	"github.com/HannesOberreiter/gbif-extinct/internal"
	"github.com/HannesOberreiter/gbif-extinct/pkg"
	"github.com/a-h/templ"
	"github.com/go-co-op/gocron/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var scheduler gocron.Scheduler

func main() {

	e := echo.New()

	/* Middleware */
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	/* Routes */
	e.GET("/", index)
	e.GET("/about", about)
	e.GET("/table", table)
	e.GET("/fetch", fetch)
	e.File("/favicon.ico", "./assets/favicon.png")
	e.Static("/assets", "./assets")

	/* Middleware */
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		OnTimeoutRouteErrorHandler: func(err error, c echo.Context) {
			slog.Warn("Timeout", "path", c.Path())
		},
		Timeout: 60 * 60 * time.Second,
	}))

	/* Start cron scheduler */
	setupScheduler()
	scheduler.Start()

	/* Start http server */
	go func() {
		if err := e.Start(":1323"); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := scheduler.Shutdown(); err != nil {
		e.Logger.Fatal(err)
	}
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

}

/* Pages */
func index(c echo.Context) error {
	var payload internal.Payload
	err := c.Bind(&payload)
	if err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}

	return render(c,
		http.StatusAccepted,
		components.PageTable(internal.GetTableData(payload), payload, internal.GetCounts(payload)))

}

func about(c echo.Context) error {
	return render(c,
		http.StatusAccepted,
		components.PageAbout())
}

/* Partials */
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
		components.Table(table, payload, internal.GetCounts(payload)))
}

/* Actions */
func fetch(c echo.Context) error {
	id := c.QueryParam("taxonID")
	if id == "" {
		c.Response().Header().Set("HX-Trigger", `{"showMessage":{"level" : "error", "message" : "Missing required taxonID QueryParam."}}`)
		return c.String(http.StatusBadRequest, "bad request")
	}

	updated := internal.UpdateLastFetchStatus(id)
	if !updated {
		c.Response().Header().Set("HX-Trigger", `{"showMessage":{"level" : "error", "message" : "Failed to update the taxa, the ID could be missing in our database."}}`)
		return c.String(http.StatusBadRequest, "Failed to update taxa")
	}

	res := pkg.FetchLatest(id)
	if res == nil {
		c.Response().Header().Set("HX-Trigger", `{"showMessage":{"level" : "error", "message" : "Timeout or no data found on GBIF for this taxon."}}`)
		return c.String(http.StatusNotFound, "No data found")
	}

	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	conn, err := internal.DB.Conn(ctx)
	if err != nil {
		slog.Error("Failed to create connection", err)
		cancel()
		return c.String(http.StatusInternalServerError, "Failed database connection")
	}
	defer conn.Close()
	var results [][]pkg.LatestObservation
	results = append(results, res)
	internal.SaveObservation(append(results, res), conn, ctx)
	cancel()

	c.Response().Header().Set("HX-Trigger", "filterSubmit")
	return c.String(http.StatusOK, "Updated")
}

// Setup cron scheduler
func setupScheduler() {
	slog.Info("Init Scheduler")

	var err error
	scheduler, err = gocron.NewScheduler()
	if err != nil {
		slog.Error("Failed to create scheduler", err)
	}

	j, err := scheduler.NewJob(
		gocron.DurationJob(
			5*60*time.Second,
		),
		gocron.NewTask(cronFetch),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		slog.Error("Failed to create job", err)
	}
	slog.Info("Job created", "job", j.ID())
}

// Fetch outdated observations and update according to latest data
func cronFetch() {
	slog.Info("Starting cron")

	ids := internal.GetOutdatedObservations()
	var results [][]pkg.LatestObservation
	for _, id := range ids {
		internal.UpdateLastFetchStatus(id)
		res := pkg.FetchLatest(id)
		if res == nil {
			continue
		}
		results = append(results, res)
	}

	if len(results) == 0 {
		slog.Info("No new observations found")
		return
	}

	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	conn, err := internal.DB.Conn(ctx)
	if err != nil {
		slog.Error("Failed to create connection", err)
		cancel()
		return
	}
	defer conn.Close()

	internal.SaveObservation(results, conn, ctx)
	cancel()
}

// Utility function to render a template
func render(c echo.Context, status int, t templ.Component) error {
	c.Response().Writer.WriteHeader(status)
	err := t.Render(context.Background(), c.Response().Writer)
	if err != nil {
		return c.String(http.StatusInternalServerError, "failed to render response template")
	}
	return nil
}
