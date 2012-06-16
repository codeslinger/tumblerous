// vim:set ts=2 sw=2 et ai ft=go:
package main

import (
  "flag"
  "runtime"
)

var (
  host = flag.String("host", "127.0.0.1", "host address on which to listen")
  port = flag.Int("port", 9999, "port on which to listen")
  template_path = flag.String("templates", "/var/www", "path to template files")
)

func main() {
  flag.Parse()
  runtime.GOMAXPROCS(runtime.NumCPU())
  app := NewApp(*host, *port, *template_path)
  app.Run()
}

