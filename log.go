// vim:set ts=2 sw=2 et ai ft=go:
package main

import (
  "fmt"
  "io"
  "os"
  "runtime"
  "strings"
  "sync"
  "time"
)

// --- LOG LEVELS -----------------------------------------------------------

type Level int

const (
  TRACE Level = iota
  DEBUG
  INFO
  WARN
  ERROR
  CRITICAL
)

var levelStrings = [...]string {"T", "D", "I", "W", "E", "C"}

func (l Level) String() string {
  if l < 0 || int(l) > len(levelStrings) {
    return "UNKNOWN"
  }
  return levelStrings[int(l)]
}

// --- LOGGER API -----------------------------------------------------------

type Logger struct {
  mutex     sync.Mutex
  level     Level
  sink      io.Writer
  buf       []byte
  callDepth int
}

func NewLogger(out io.Writer, lvl Level, callDepth int) *Logger {
  return &Logger{sink: out, level: lvl, callDepth: callDepth}
}

func (log *Logger) GetLevel() Level {
  return log.level
}

func (log *Logger) SetLevel(lvl Level) {
  if lvl < TRACE || lvl > CRITICAL {
    return
  }
  log.level = lvl
}

func (log *Logger) Trace(arg0 interface{}, args ...interface{}) {
  switch first := arg0.(type) {
  case string:
    log.logf(TRACE, first, args...)
  case func() string:
    log.logc(TRACE, first)
  default:
    log.logf(TRACE,
             fmt.Sprint(arg0) + strings.Repeat(" %v", len(args)), args...)
  }
}

func (log *Logger) Debug(arg0 interface{}, args ...interface{}) {
  switch first := arg0.(type) {
  case string:
    log.logf(DEBUG, first, args...)
  case func() string:
    log.logc(DEBUG, first)
  default:
    log.logf(DEBUG,
             fmt.Sprint(arg0) + strings.Repeat(" %v", len(args)), args...)
  }
}

func (log *Logger) Info(arg0 interface{}, args ...interface{}) {
  switch first := arg0.(type) {
  case string:
    log.logf(INFO, first, args...)
  case func() string:
    log.logc(INFO, first)
  default:
    log.logf(INFO,
             fmt.Sprint(arg0) + strings.Repeat(" %v", len(args)), args...)
  }
}

func (log *Logger) Warn(arg0 interface{}, args ...interface{}) {
  switch first := arg0.(type) {
  case string:
    log.logf(WARN, first, args...)
  case func() string:
    log.logc(WARN, first)
  default:
    log.logf(WARN,
             fmt.Sprint(arg0) + strings.Repeat(" %v", len(args)), args...)
  }
}

func (log *Logger) Error(arg0 interface{}, args ...interface{}) {
  switch first := arg0.(type) {
  case string:
    log.logf(ERROR, first, args...)
  case func() string:
    log.logc(ERROR, first)
  default:
    log.logf(ERROR,
             fmt.Sprint(arg0) + strings.Repeat(" %v", len(args)), args...)
  }
}

func (log *Logger) Critical(arg0 interface{}, args ...interface{}) {
  msg := ""
  switch first := arg0.(type) {
  case string:
    msg = log.logf(CRITICAL, first, args...)
  case func() string:
    msg = log.logc(CRITICAL, first)
  default:
    msg = log.logf(
            CRITICAL,
            fmt.Sprint(arg0) + strings.Repeat(" %v", len(args)), args...)
  }
  panic(msg)
}

// --- SINGLETON INTERFACE ---------------------------------------------------

var dfl = NewLogger(os.Stderr, TRACE, 3)

func GetLevel() Level {
  return dfl.GetLevel()
}

func SetLevel(lvl Level) {
  dfl.SetLevel(lvl)
}

func Trace(arg0 interface{}, args ...interface{}) {
  dfl.Trace(arg0, args...)
}

func Debug(arg0 interface{}, args ...interface{}) {
  dfl.Debug(arg0, args...)
}

func Info(arg0 interface{}, args ...interface{}) {
  dfl.Info(arg0, args...)
}

func Warn(arg0 interface{}, args ...interface{}) {
  dfl.Warn(arg0, args...)
}

func Error(arg0 interface{}, args ...interface{}) {
  dfl.Error(arg0, args...)
}

func Critical(arg0 interface{}, args ...interface{}) {
  dfl.Critical(arg0, args...)
}

// --- CLOSURE FACTORY ------------------------------------------------------

func Closure(format string, args ...interface{}) func() string {
  return func() string {
    return fmt.Sprintf(format, args...)
  }
}

// --- INTERNAL FUNCTIONS ---------------------------------------------------

func (log *Logger) logf(lvl Level, format string, args ...interface{}) string {
  if lvl < log.level {
    return ""
  }
  _, file, line, ok := runtime.Caller(log.callDepth)
  if !ok {
    file = "???"
    line = 0
  }
  msg := format
  if len(args) > 0 {
    msg = fmt.Sprintf(format, args...)
  }
  log.write(lvl, time.Now(), file, line, msg)
  return msg
}

func (log *Logger) logc(lvl Level, closure func() string) string {
  if lvl < log.level {
    return ""
  }
  _, file, line, ok := runtime.Caller(log.callDepth)
  if !ok {
    file = "???"
    line = 0
  }
  msg := closure()
  log.write(lvl, time.Now(), file, line, msg)
  return msg
}

func (log *Logger) write(lvl Level, now time.Time, file string, line int, msg string) error {
  log.mutex.Lock()
  defer log.mutex.Unlock()
  log.buf = log.buf[:0]
  log.fmtPrefix(&log.buf, lvl, now, file, line)
  log.buf = append(log.buf, msg...)
  if len(msg) > 0 && msg[len(msg) - 1] != '\n' {
    log.buf = append(log.buf, '\n')
  }
  _, err := log.sink.Write(log.buf)
  return err
}

func (log *Logger) fmtPrefix(buf *[]byte, lvl Level, t time.Time, file string, line int) {
  hdr := fmt.Sprintf("%s [%s %d] (%s:%d) ",
                     levelStrings[int(lvl)],
                     t.Format(time.RFC3339),
                     os.Getpid(),
                     log.fileBasename(file),
                     line)
  *buf = append(*buf, hdr...)
}

func (log *Logger) fileBasename(file string) string {
  short := file
  for i := len(file) - 1; i > 0; i-- {
    if file[i] == '/' {
      short = file[i+1:]
      break
    }
  }
  return short
}

