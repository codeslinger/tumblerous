// vim:set ts=2 sw=2 et ai ft=go:
package main

import (
  "flag"
  "runtime"
)

var (
  host          string
  port          int
  template_path string
)

func init() {
  flag.StringVar(&host, "host", "127.0.0.1", "host address on which to listen")
  flag.IntVar(&port, "port", 9999, "port on which to listen")
  flag.StringVar(&template_path, "templates", "/var/www", "path to template files")
  flag.Parse()
}

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())
  app := NewApp(host, port, template_path, INFO)
  app.Run()
}

