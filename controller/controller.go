package controller

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"

    "github.com/attilamonteiro/deserialize-explosion/service"
)

type Controller struct {
    svc *service.Service
}

func NewController(s *service.Service) *Controller {
    return &Controller{svc: s}
}

func (c *Controller) Register(mux *http.ServeMux) {
    mux.HandleFunc("/receitas", c.Receitas)
    mux.HandleFunc("/bench", c.Bench)
    mux.HandleFunc("/clear-cache", c.ClearCache)
    mux.HandleFunc("/create_perpage_cache", c.CreatePerPageCache)
}

func (c *Controller) respondJSON(w http.ResponseWriter, v interface{}) {
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(v)
}

func (c *Controller) Receitas(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    page, _ := strconv.Atoi(q.Get("page"))
    if page <= 0 {
        page = 1
    }
    pageSize := atoiOrDefault(q.Get("pageSize"), 500)
    num := atoiOrDefault(q.Get("num"), 50000)
    dbDelay := atoiOrDefault(q.Get("dbDelay"), 30)

    items, err := c.svc.GetPageLegacy(page, pageSize, num, dbDelay)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    c.respondJSON(w, items)
}

func (c *Controller) Bench(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    num := atoiOrDefault(q.Get("num"), 1000)
    pageSize := atoiOrDefault(q.Get("pageSize"), 100)
    dbDelay := atoiOrDefault(q.Get("dbDelay"), 2)
    rep, _ := c.svc.Bench(num, pageSize, dbDelay)
    c.respondJSON(w, rep)
}

func (c *Controller) ClearCache(w http.ResponseWriter, r *http.Request) {
    c.svc.Repo().Clear()
    fmt.Fprintln(w, "cache cleared")
}

func (c *Controller) CreatePerPageCache(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    num := atoiOrDefault(q.Get("num"), 1000)
    pageSize := atoiOrDefault(q.Get("pageSize"), 100)
    // delegate to bench workflow: build per-page by invoking Bench which builds per-page when cleared
    _, _ = c.svc.Bench(num, pageSize, 0)
    fmt.Fprintf(w, "per-page cache created for %d items pageSize=%d\n", num, pageSize)
}

// small helper used locally to parse ints
func atoiOrDefault(s string, def int) int {
    if s == "" {
        return def
    }
    v, err := strconv.Atoi(s)
    if err != nil {
        return def
    }
    return v
}
