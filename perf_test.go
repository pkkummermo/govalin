package govalin

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// perfGateEnv turns the allocation gate on. It is off by default because the
// budgets below hold for one Go version on one platform at a time: escape
// analysis and inlining change between releases, and the static file shape
// allocates differently per OS, so a run elsewhere fails on a difference that is
// not a regression. CI runs this on the baseline toolchain on ubuntu, which is
// where the numbers come from, the same way race detection is scoped.
const perfGateEnv = "GOVALIN_PERF_GATE"

// allocationBudget is what a request shape is allowed to allocate.
//
// The numbers are measured, not aimed for: they are what the code does today.
// Raising one is a decision to take deliberately, in the diff that needs it,
// with the reason in the commit — not a number to nudge until the gate passes.
type allocationBudget struct {
	name    string
	target  string
	allowed int
	build   func(app *App)
}

var allocationBudgets = []allocationBudget{
	{
		name:    "text",
		target:  "/text",
		allowed: 8,
		build:   func(app *App) { app.Get("/text", func(call *Call) { call.Text("Hello world") }) },
	},
	{
		name:    "json",
		target:  "/json",
		allowed: 9,
		build: func(app *App) {
			type payload struct {
				Name  string `json:"name"`
				Count int    `json:"count"`
			}

			app.Get("/json", func(call *Call) { call.JSON(payload{Name: "govalin", Count: 42}) })
		},
	},
	{
		name:    "status only",
		target:  "/status",
		allowed: 5,
		build:   func(app *App) { app.Get("/status", func(call *Call) { call.Status(http.StatusNoContent) }) },
	},
	{
		name:    "before and after handlers",
		target:  "/text",
		allowed: 8,
		build: func(app *App) {
			app.Before("/*", func(_ *Call) bool { return true })
			app.Get("/text", func(call *Call) { call.Text("Hello world") })
			app.After("/*", func(_ *Call) {})
		},
	},
	{
		name:    "not found",
		target:  "/missing",
		allowed: 33,
		build:   func(app *App) { app.Get("/text", func(call *Call) { call.Text("Hello world") }) },
	},
	{
		name:    "raw handler",
		target:  "/raw",
		allowed: 6,
		build: func(app *App) {
			app.HTTPServe("/raw", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("Hello world"))
			})
		},
	},
	{
		name:    "static file",
		target:  "/static/sub/test.html",
		allowed: 39,
		build: func(app *App) {
			app.Static("/static", func(_ *Call, staticConfig *StaticConfig) {
				staticConfig.WithStaticPath("internal/testdata/static")
			})
		},
	},
}

// TestAllocationBudget is the performance gate.
//
// It counts allocations rather than timing anything: an allocation count is the
// same on every run of a given build, while wall-clock time on a shared CI
// runner is not, and a gate that fails at random gets switched off. Per-request
// allocations are also what an HTTP framework's overhead actually consists of.
func TestAllocationBudget(t *testing.T) {
	if os.Getenv(perfGateEnv) == "" {
		t.Skipf("performance gate is off; set %s=1 to run it", perfGateEnv)
	}

	for _, budget := range allocationBudgets {
		t.Run(budget.name, func(t *testing.T) {
			app := benchApp(budget.build)
			request := httptest.NewRequest(http.MethodGet, budget.target, nil)
			writer := &discardWriter{header: http.Header{}}

			allocations := int(testing.AllocsPerRun(200, func() {
				writer.reset()
				app.rootHandlerFunc(writer, request)
			}))

			if allocations > budget.allowed {
				t.Errorf(
					"%s allocates %d times per request, budget is %d.\n"+
						"If the extra allocation is intended, raise the budget in perf_test.go "+
						"in the same commit and say why.",
					budget.name, allocations, budget.allowed,
				)
			}

			t.Logf("%s: %d allocations per request (budget %d)", budget.name, allocations, budget.allowed)
		})
	}
}
