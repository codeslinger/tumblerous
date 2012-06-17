// vim:set ts=2 sw=2 et ai ft=go:
package main

import (
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
  flag.Parse()
}

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())
  app := NewWebapp(host, port, NewLogger(os.Stdout, INFO))
  app.Run()
}

