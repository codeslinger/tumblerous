// vim:set ts=2 sw=2 et ai ft=go:
package main

import (
  "bytes"
  "fmt"
  "net/http"
  "os"
  "regexp"
  "runtime"
  "strconv"
  "strings"
  "time"
)

// --- REQUEST API ----------------------------------------------------------

type Request struct {
  w             http.ResponseWriter
  r             *http.Request
  app           *App
  status        int
  contentLength int
  contentType   string
  date          time.Time
  replied       bool
}

func (req *Request) SetHeader(name, val string) {
  req.w.Header().Set(name, val)
}

func (req *Request) AddHeader(name, val string) {
  req.w.Header().Add(name, val)
}

func (req *Request) OK(body string) {
  req.Reply(http.StatusOK, body)
}

func (req *Request) NotFound(body string) {
  req.Reply(http.StatusNotFound, body)
}

func (req *Request) Reply(status int, body string) {
  if req.replied {
    req.app.Log.Critical("this context has already been replied to!")
  }
  req.status = status
  req.contentLength = len(body)
  req.SetHeader("Date", req.httpDate(req.date))
  if req.contentLength > 0 {
    req.SetHeader("Content-Type", req.contentType)
    req.SetHeader("Content-Length", strconv.Itoa(req.contentLength))
  }
  if req.status >= 400 {
    req.SetHeader("Connection", "close")
  }
  req.replied = true
  req.w.WriteHeader(req.status)
  if req.contentLength > 0 {
    req.w.Write([]byte(body))
  }
}

// --- REQUEST INTERNALS ----------------------------------------------------

func newRequest(w http.ResponseWriter, r *http.Request, app *App) *Request {
  req := &Request {
    w:             w,
    r:             r,
    app:           app,
    status:        200,
    contentLength: 0,
    contentType:   "text/html; charset=utf-8",
    date:          time.Now(),
    replied:       false,
  }
  return req
}

func (req *Request) logHit() {
  bytesSent := "-"
  if req.contentLength > 0 {
    bytesSent = strconv.Itoa(req.contentLength)
  }
  req.app.Log.Info("hit: %s %s %s %s %d %s\n",
                   req.r.RemoteAddr,
                   req.r.Method,
                   req.r.URL.Path,
                   req.r.Proto,
                   req.status,
                   bytesSent)
}

func (req *Request) httpDate(t time.Time) string {
  f := t.UTC().Format(time.RFC1123)
  if strings.HasSuffix(f, "UTC") {
    f = f[0:len(f)-3] + "GMT"
  }
  return f
}

// --- APP API --------------------------------------------------------------

type RouteHandler func(*Request, []string)

type route struct {
  pattern string
  re      *regexp.Regexp
  method  string
  handler RouteHandler
}

type App struct {
  Log          *Logger
  LogHits      bool
  host         string
  port         int
  templatePath string
  routes       []route
}

func NewApp(host string, port int, templatePath string, lvl Level) *App {
  app := &App {
    Log:          NewLogger(os.Stdout, lvl, 2),
    LogHits:      true,
    host:         host,
    port:         port,
    templatePath: templatePath,
  }
  return app
}

func (app *App) Run() {
  addr := fmt.Sprintf("%s:%d", app.host, app.port)
  s := &http.Server {
    Addr:           addr,
    Handler:        app,
    ReadTimeout:    10 * time.Second,
    WriteTimeout:   10 * time.Second,
    MaxHeaderBytes: 1 << 20,
  }
  app.Log.Info("application started: listening on %s", addr)
  err := s.ListenAndServe()
  if err != nil {
    app.Log.Error(err)
  }
}

// --- ROUTE REGISTRATION ---------------------------------------------------

func (app *App) Get(pattern string, handler RouteHandler) {
  app.registerRoute(pattern, "GET", handler)
}

func (app *App) Post(pattern string, handler RouteHandler) {
  app.registerRoute(pattern, "POST", handler)
}

func (app *App) Put(pattern string, handler RouteHandler) {
  app.registerRoute(pattern, "PUT", handler)
}

func (app *App) Delete(pattern string, handler RouteHandler) {
  app.registerRoute(pattern, "DELETE", handler)
}

// --- APP INTERNALS --------------------------------------------------------

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  req := newRequest(w, r, app)
  path := r.URL.Path
  for i := 0; i < len(app.routes); i++ {
    route := app.routes[i]
    if r.Method != route.method && !(r.Method == "HEAD" && route.method == "GET") {
      continue
    }
    if !route.re.MatchString(path) {
      continue
    }
    match := route.re.FindStringSubmatch(path)
    err := app.protect(route.handler, req, match[1:])
    if err != nil {
      req.Reply(500, "Internal server error")
    }
    return
  }
  req.NotFound("<h1>Not found</h1>")
  if req.app.LogHits {
    req.logHit()
  }
}

func (app *App) registerRoute(pattern string, method string, handler RouteHandler) {
  re, err := regexp.Compile(pattern)
  if err != nil {
    app.Log.Critical("could not compile route pattern: %q", pattern)
  }
  app.routes = append(app.routes, route{pattern, re, method, handler})
}

func (app *App) protect(handler RouteHandler, req *Request, args []string) (e interface{}) {
  defer func() {
    if err := recover(); err != nil {
      e = err
      var buf bytes.Buffer
      fmt.Fprintf(&buf, "handler crashed: %v\n", err)
      for i := 2; ; i++ {
        _, file, line, ok := runtime.Caller(i)
        if !ok {
          break
        }
        fmt.Fprintf(&buf, "! %s:%d\n", file, line)
      }
      app.Log.Error(buf.String())
    }
  }()
  handler(req, args)
  return
}

