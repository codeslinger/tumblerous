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

// The Request record encapsulates all the of the state required to handle
// an HTTP request/response cycle.
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

// Sets the named header to the given value. This will override any existing
// value for the named header with the given value.
func (req *Request) SetHeader(name, val string) {
  req.w.Header().Set(name, val)
}

// Add an instance of a header to the output with the given value. This allows
// for more than one header with the given name to be output (e.g. Set-Cookie).
func (req *Request) AddHeader(name, val string) {
  req.w.Header().Add(name, val)
}

// Respond to the request with an HTTP OK (200) status code and the given
// response body. Use an empty string for no body.
func (req *Request) OK(body string) {
  req.Reply(http.StatusOK, body)
}

// Respond to the request with an HTTP Not Found (404) status and the given
// response body. Use an empty string for no body.
func (req *Request) NotFound(body string) {
  req.Reply(http.StatusNotFound, body)
}

// Respond to the request with the given status code and response body. Use
// an empty string for no body.
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

// Private constructor for Request records. These should only be created by
// an App instance.
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

// Record pertinent request and response information in the log.
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

// Format a given time for use with HTTP headers.
func (req *Request) httpDate(t time.Time) string {
  f := t.UTC().Format(time.RFC1123)
  if strings.HasSuffix(f, "UTC") {
    f = f[0:len(f)-3] + "GMT"
  }
  return f
}

// --- APP API --------------------------------------------------------------

// The RouteHandler is the type a function should be if it wishes to register
// for handling a route.
//
// If a request arrives matching the pattern for a route, its RouteHandler will
// be called to respond to the request. The RouteHandler func is given a
// pointer to a Request record and a list of argument values extracted from the
// route pattern given.
//
// E.g. if a route is registered with the pattern: "/foo/(\d+)/bar/(\w+)" Then
// args will contain two values, the first being the string matched between the
// "foo" and the "bar" parts of the request URI and the second being the string
// matched between the "bar" and the end of the string.
type RouteHandler func(*Request, []string)

type route struct {
  pattern string
  re      *regexp.Regexp
  method  string
  handler RouteHandler
}

// An App is the main edifice for a web application.
type App struct {
  Log          *Logger
  LogHits      bool
  host         string
  port         int
  templatePath string
  routes       []route
}

// Create a new App instance. The host and port on which to listen are given,
// as is the path to any templates the application will need, as well as the
// minimum log level for messages output to the log.
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

// Start the App listening and serving requests.
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

// Register a route for a given pattern for GET requests. (will also be called
// for HEAD requests)
func (app *App) Get(pattern string, handler RouteHandler) {
  app.registerRoute(pattern, "GET", handler)
}

// Register a route for a given pattern for POST requests.
func (app *App) Post(pattern string, handler RouteHandler) {
  app.registerRoute(pattern, "POST", handler)
}

// Register a route for a given pattern for PUT requests.
func (app *App) Put(pattern string, handler RouteHandler) {
  app.registerRoute(pattern, "PUT", handler)
}

// Register a route for a given pattern for DELETE requests.
func (app *App) Delete(pattern string, handler RouteHandler) {
  app.registerRoute(pattern, "DELETE", handler)
}

// --- APP INTERNALS --------------------------------------------------------

// Main callback for App instance on receipt of new HTTP request.
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

// Does the work of registering a route pattern and handler with this
// App instance.
func (app *App) registerRoute(pattern string, method string, handler RouteHandler) {
  re, err := regexp.Compile(pattern)
  if err != nil {
    app.Log.Critical("could not compile route pattern: %q", pattern)
  }
  app.routes = append(app.routes, route{pattern, re, method, handler})
}

// Run a RouteHandler safely, ensuring that panics inside handlers are trapped
// and logged.
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

