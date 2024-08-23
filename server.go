package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/HannesOberreiter/gbif-extinct/components"
	"github.com/HannesOberreiter/gbif-extinct/internal"
	"github.com/HannesOberreiter/gbif-extinct/pkg/gbif"
	"github.com/HannesOberreiter/gbif-extinct/pkg/queries"
	"github.com/a-h/templ"
	"github.com/go-co-op/gocron/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var scheduler gocron.Scheduler
var cacheBuster = time.Now().Unix()

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
	e.GET("/download", download)
	e.File("/favicon.ico", "./assets/favicon.png")
	e.Static("/assets", "./assets")

	/* Middleware */
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		OnTimeoutRouteErrorHandler: func(err error, c echo.Context) {
			slog.Warn("Timeout", "path", c.Path())
		},
		Timeout: 15 * 60 * time.Second,
	}))

	/* Init Packages */
	internal.Load()
	internal.Migrations(internal.DB, internal.Config.ROOT) // Update the database schema to the latest version
	gbif.UpdateConfig(gbif.Config{UserAgentPrefix: internal.Config.UserAgentPrefix})
	components.RenderAbout()

	/* Start cron scheduler */
	setupScheduler()
	if scheduler != nil {
		scheduler.Start()
	}

	/* Start http server */
	go func() {
		if err := e.Start(":1323"); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	/* Graceful shutdown */
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	slog.Info("Server shutdown")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := scheduler.Shutdown(); err != nil {
		e.Logger.Fatal("Failed to stop scheduler", "error", err)
	}
	slog.Info("Scheduler stopped")
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal("Failed to stop server", "error", err)
	}
	slog.Info("Server stopped")
	if err := internal.DB.Close(); err != nil {
		slog.Error("Failed to close database", "error", err)
	}
	slog.Info("Database closed")

}

type Payload struct {
	ORDER_BY      *string `query:"order_by"`
	ORDER_DIR     *string `query:"order_dir"`
	SEARCH        *string `query:"search"`
	COUNTRY       *string `query:"country"`
	RANK          *string `query:"rank"`
	TAXA          *string `query:"taxa"`
	PAGE          *string `query:"page"`
	SHOW_SYNONYMS *bool   `query:"show_synonyms"`
}

/* Pages */
func index(c echo.Context) error {
	q := buildQuery(c)
	counts := q.GetCounts(internal.DB)

	return render(c,
		http.StatusAccepted,
		components.PageTable(q.GetTableData(internal.DB), q, counts, components.CalculatePages(counts, q), cacheBuster))

}

func about(c echo.Context) error {
	countTaxa := queries.GetCountTotalTaxa(internal.DB)
	countLastFetched := queries.GetCountFetchedLastTwelveMonths(internal.DB)

	return render(c,
		http.StatusAccepted,
		components.PageAbout(countTaxa, countLastFetched, cacheBuster))
}

/* Partials */
func table(c echo.Context) error {
	q := buildQuery(c)
	querystring := c.QueryString()

	table := q.GetTableData(internal.DB)
	counts := q.GetCounts(internal.DB)

	if c.Request().Header.Get("HX-Request") == "" {
		return c.JSON(http.StatusOK, table)
	}

	c.Response().Header().Set("HX-Push-Url", "/?"+querystring)
	return render(c,
		http.StatusAccepted,
		components.Table(table, q, counts, components.CalculatePages(counts, q)))
}

/* Actions */
func fetch(c echo.Context) error {

	id := c.QueryParam("taxonID")
	if id == "" {
		c.Response().Header().Set("HX-Trigger", `{"showMessage":{"level" : "error", "message" : "Missing required taxonID QueryParam."}}`)
		return c.String(http.StatusBadRequest, "bad request")
	}
	var err error
	var synonymId string

	synonymId, err = gbif.GetSynonymID(internal.DB, id)
	if err != nil {
		c.Response().Header().Set("HX-Trigger", `{"showMessage":{"level" : "error", "message" : "Failed to get SynonymID"}}`)
		return c.String(http.StatusBadRequest, "Failed to get SynonymID")
	}

	updated := gbif.UpdateLastFetchStatus(internal.DB, synonymId)
	if !updated {
		c.Response().Header().Set("HX-Trigger", `{"showMessage":{"level" : "error", "message" : "Failed to update the taxa, the ID could be missing in our database."}}`)
		return c.String(http.StatusBadRequest, "Failed to update taxa")
	}

	res := gbif.FetchLatest(synonymId)
	if res == nil {
		c.Response().Header().Set("HX-Trigger", `{"showMessage":{"level" : "error", "message" : "Timeout or no data found on GBIF for this taxon."}}`)
		return c.String(http.StatusNotFound, "No data found")
	}

	var results = &[][]gbif.LatestObservation{}
	*results = append(*results, *res)
	gbif.SaveObservation(results, internal.DB)

	c.Response().Header().Set("HX-Trigger", "filterSubmit")
	return c.String(http.StatusOK, "Updated")
}

// Download data as CSV
func download(c echo.Context) error {

	q := buildQuery(c)
	table := q.GetTableData(internal.DB, true)
	csv := table.CreateCSV()

	filename := fmt.Sprintf("extinct-%s-%s.csv", time.Now().Format("2006-01-02"), strings.ReplaceAll(c.QueryString(), "&", "-"))

	c.Response().Header().Set("Content-Disposition", "attachment; filename="+filename)
	c.Response().Header().Set("Content-Type", "text/csv")
	return c.String(http.StatusOK, csv)
}

// Setup cron scheduler
func setupScheduler() {
	interval := internal.Config.CronJobIntervalSec
	if interval == 0 {
		slog.Info("Scheduler disabled", "interval", interval)
		return
	}
	slog.Info("Init Scheduler")

	var err error
	scheduler, err = gocron.NewScheduler()
	if err != nil {
		slog.Error(err.Error())
	}

	j, err := scheduler.NewJob(
		gocron.DurationJob(time.Duration(interval)*time.Second),
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

	ids := gbif.GetOutdatedObservations(internal.DB)
	var results = &[][]gbif.LatestObservation{}
	for _, id := range ids {
		gbif.UpdateLastFetchStatus(internal.DB, id)
		res := gbif.FetchLatest(id)
		if res == nil {
			continue
		}
		*results = append(*results, *res)
	}

	if len(*results) == 0 {
		slog.Info("No new observations found")
		return
	}

	gbif.SaveObservation(results, internal.DB)
}

// Utility function to render a template
func render(c echo.Context, status int, t templ.Component) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 25*time.Second)
	defer cancel()
	c.Response().Writer.WriteHeader(status)
	err := t.Render(ctx, c.Response().Writer)
	if err != nil {
		return c.String(http.StatusInternalServerError, "failed to render response template")
	}
	return nil
}

// Utility function to build a query struct with sane and clean defaults from the payload parser
func buildQuery(c echo.Context) queries.Query {
	var payload Payload
	err := c.Bind(&payload)
	if err != nil {
		slog.Warn("Failed to bind payload", "error", err)
		return queries.NewQuery(nil)
	}
	return queries.NewQuery(payload)
}
