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

// Priority level of log message.
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

// String representation of priority level.
func (l Level) String() string {
  if l < 0 || int(l) > len(levelStrings) {
    return "UNKNOWN"
  }
  return levelStrings[int(l)]
}

// --- LOGGER API -----------------------------------------------------------

// Logger allows for serialized output to a sink of log messages, filtering
// any incoming messages below the specified priority level. The Logger is
// safe to use from concurrent goroutines.
type Logger struct {
  mutex     sync.Mutex
  level     Level
  sink      io.Writer
  buf       []byte
  callDepth int
}

// Create a new Logger. The callDepth param refers to the number of stack
// frames to ignore when determining the file/line of the calling function.
// Typically, this should be 2.
func NewLogger(out io.Writer, lvl Level, callDepth int) *Logger {
  return &Logger{sink: out, level: lvl, callDepth: callDepth}
}

// Get the lowest priority level for which this Logger will emit messages.
func (log *Logger) GetLevel() Level {
  return log.level
}

// Set the lowest priority level for which this Logger will emit messages.
func (log *Logger) SetLevel(lvl Level) {
  if lvl < TRACE || lvl > CRITICAL {
    return
  }
  log.level = lvl
}

// Log a TRACE level message to this Logger.
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

// Log a DEBUG level message to this Logger.
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

// Log an INFO level message to this Logger.
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

// Log a WARN level message to this Logger.
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

// Log an ERROR level message to this Logger.
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

// Log a CRITICAL level message to this Logger. This will also generate a panic
// with the log message after the message has been written to the log.
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

// The default logger will write to STDERR and emit messages for all priority
// levels.
var dfl = NewLogger(os.Stderr, TRACE, 3)

// Get the lowest priority level for which the default logger will emit a
// message.
func GetLevel() Level {
  return dfl.GetLevel()
}

// Set the lowest priority level for which the default logger will emit a
// message.
func SetLevel(lvl Level) {
  dfl.SetLevel(lvl)
}

// Log a TRACE level message to the default logger.
func Trace(arg0 interface{}, args ...interface{}) {
  dfl.Trace(arg0, args...)
}

// Log a TRACE level message to the default logger.
func Debug(arg0 interface{}, args ...interface{}) {
  dfl.Debug(arg0, args...)
}

// Log a TRACE level message to the default logger.
func Info(arg0 interface{}, args ...interface{}) {
  dfl.Info(arg0, args...)
}

// Log a TRACE level message to the default logger.
func Warn(arg0 interface{}, args ...interface{}) {
  dfl.Warn(arg0, args...)
}

// Log a TRACE level message to the default logger.
func Error(arg0 interface{}, args ...interface{}) {
  dfl.Error(arg0, args...)
}

// Log a CRITICAL level message to the default logger. This will also
// generate a panic with the log message after the message has been written
// to the log.
func Critical(arg0 interface{}, args ...interface{}) {
  dfl.Critical(arg0, args...)
}

// --- CLOSURE FACTORY ------------------------------------------------------

// Build a closure to generate a log message. This is used to defer
// potentially expensive computation in the case where the log message is
// not likely to be emitted given its low priority level.
func Closure(format string, args ...interface{}) func() string {
  return func() string {
    return fmt.Sprintf(format, args...)
  }
}

// --- INTERNAL FUNCTIONS ---------------------------------------------------

// Log a message via format string and args.
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

// Log a message via a call to a closure.
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

// Write a message to the log sink.
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

// Concatenate the log message prefix to the given byte array.
func (log *Logger) fmtPrefix(buf *[]byte, lvl Level, t time.Time, file string, line int) {
  hdr := fmt.Sprintf("%s [%s %d] (%s:%d) ",
                     levelStrings[int(lvl)],
                     t.Format(time.RFC3339),
                     os.Getpid(),
                     log.fileBasename(file),
                     line)
  *buf = append(*buf, hdr...)
}

// Determine a the base name of a file. (i.e. shortname)
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

