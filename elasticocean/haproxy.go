package elasticocean

import (
  "io/ioutil"
  "os"
  "os/exec"
  "strings"
  "syscall"
)

func (e *Elastic) loadhaproxyConf() (err error) {
  e.FileLock.Lock()
  defer e.FileLock.Unlock()
  var b []byte
  if b, err = ioutil.ReadFile(e.Haproxy.Config); err != nil {
    return err
  }
  st := string(b)
  sep := strings.SplitAfter(st, "\n")
  var server bool
  for _, line := range sep {
    if strings.Contains(line, "#GOEDIT") {
      switch strings.Contains(line, "Start") {
      case true:
        server = true
      case false:
        server = false
      }
      continue
    }

    if server {
      s := e.NewServer(line)
      e.Servers[s.Name] = s
    }
  }
  return
}

func (e Elastic) writeConf() (err error) {
  var b []byte
  if b, err = ioutil.ReadFile(e.Haproxy.Config); err != nil {
    return err
  }
  st := string(b)
  sep := strings.SplitAfter(st, "\n")
  var server bool
  out := ""
  for _, line := range sep {
    if strings.Contains(line, "#GOEDIT") {
      switch strings.Contains(line, "Start") {
      case true:
        server = true
        out = e.insertServers(out)
      case false:
        server = false
      }
      continue
    }

    switch server {
    case true:
      continue
    case false:
      out += line
    }
  }
  return ioutil.WriteFile(e.Haproxy.Config, []byte(out), os.ModeAppend)
}

func (e Elastic) insertServers(out string) string {
  out += "  #GOEDIT Start server list\n"
  for _, server := range e.Servers {
    out += server.ConfLine()
  }
  out += "  #GOEDIT End server list\n"
  return out
}

func (e *Elastic) AddSlave(s *Server) (err error) {
  e.FileLock.Lock()
  defer e.FileLock.Unlock()
  file, err := os.OpenFile(e.Haproxy.Config, os.O_APPEND, 0666)
  if err != nil {
    return err
  }

  if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
    return err
  }
  defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
  // TODO Map lock or write loop
  e.Servers[s.Name] = s
  e.writeConf()
  if err = Restart(); err != nil {
    err = Restart()
  }
  return
}

func Restart() (err error) {
  restart := exec.Command("service", "haproxy", "restart")
  return restart.Run()
}
