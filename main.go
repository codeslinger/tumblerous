// vim:set ts=2 sw=2 et ai ft=go:
package main

import (
  "github.com/codeslinger/log"
  "github.com/codeslinger/webapp"
  "flag"
  "os"
  "runtime"
)

var (
  host string
  port int
)

func init() {
  flag.StringVar(&host, "host", "127.0.0.1", "host address on which to listen")
  flag.IntVar(&port, "port", 9999, "port on which to listen")
}

func main() {
  flag.Parse()
  runtime.GOMAXPROCS(runtime.NumCPU())
  logger := log.NewLogger(os.Stdout, log.INFO)
  app := webapp.NewWebapp(host, port, logger)
  app.Run()
}

