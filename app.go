// vim:set ts=2 sw=2 et ai ft=go:
package main

import (
  "fmt"
  "log"
  "net/http"
  "regexp"
  "runtime"
  "strconv"
  "strings"
  "time"
)

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

type RouteHandler func(*Request, []string)

type route struct {
  pattern string
  re      *regexp.Regexp
  method  string
  handler RouteHandler
}

type App struct {
  host         string
  port         int
  templatePath string
  routes       []route
}

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
    log.Panic("this context has already been replied to!")
  }
  req.status = status
  req.contentLength = len(body)
  req.SetHeader("Date", req.httpDate(req.date))
  if req.contentLength > 0 {
    req.SetHeader("Content-Type", req.contentType)
    req.SetHeader("Content-Length", strconv.Itoa(req.contentLength))
  }
  req.replied = true
  req.w.WriteHeader(req.status)
  if req.contentLength > 0 {
    req.w.Write([]byte(body))
  }
}

func (req *Request) logHit() {
  timestamp := req.date.Format("02/Jan/2006:15:04:05 -0700")
  bytesSent := "-"
  if req.contentLength > 0 {
    bytesSent = strconv.Itoa(req.contentLength)
  }
  fmt.Printf("%s - - [%s] \"%s %s %s\" %d %s\n",
             req.r.RemoteAddr,
             timestamp,
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

func NewApp(host string, port int, templatePath string) *App {
  app := &App {
    host: host,
    port: port,
    templatePath: templatePath,
  }
  return app
}

func (app *App) Run() {
  s := &http.Server {
    Addr:           fmt.Sprintf("%s:%d", app.host, app.port),
    Handler:        app,
    ReadTimeout:    10 * time.Second,
    WriteTimeout:   10 * time.Second,
    MaxHeaderBytes: 1 << 20,
  }
  err := s.ListenAndServe()
  if err != nil {
    log.Fatal(err)
  }
}

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
    err := app.safeRun(route.handler, req, match[1:])
    if err != nil {
      req.Reply(500, "Internal server error")
    }
    return
  }
  req.NotFound("<h1>Not found</h1>")
  req.logHit()
}

func (app *App) registerRoute(pattern string, method string, handler RouteHandler) {
  re, err := regexp.Compile(pattern)
  if err != nil {
    log.Panicf("could not compile route pattern: %q", pattern)
  }
  app.routes = append(app.routes, route{pattern, re, method, handler})
}

func (app *App) safeRun(handler RouteHandler, req *Request, args []string) (e interface{}) {
  defer func() {
    if err := recover(); err != nil {
      e = err
      log.Println("handler crashed", err)
      for i := 1; ; i++ {
        _, file, line, ok := runtime.Caller(i)
        if !ok {
          break
        }
        log.Println(file, line)
      }
    }
  }()
  handler(req, args)
  return
}

