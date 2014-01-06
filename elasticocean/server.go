package elasticocean

import (
  "fmt"
  "github.com/macb/go-haproxy/haproxy"
  "strconv"
  "strings"
)

type Server struct {
  Ip          string
  Name        string
  ImageId     int
  UID         string
  Backup      bool
  HealthCheck bool
  MaxConn     int
  Load        *haproxy.Load
}

func (e Elastic) NewServer(confLine string) *Server {
  s := new(Server)
  confLine = strings.Trim(confLine, " ")
  confLine = strings.Trim(confLine, "\n")
  parts := strings.Split(confLine, " ")
  for i, part := range parts {
    switch {
    case part == "backup":
      s.Backup = true
    case part == "check":
      s.HealthCheck = true
    case part == "maxconn":
      s.MaxConn, _ = strconv.Atoi(parts[i+1])
    case strings.Contains(part, e.BaseName):
      s.Name = part
      part = strings.Trim(part, e.BaseName)
      versionCount := strings.Split(part, "-")
      s.ImageId, _ = strconv.Atoi(versionCount[0])
      s.UID = versionCount[1]
    case strings.Contains(part, e.ServerPort):
      s.Ip = part
    }
  }
  return s
}

func (s Server) ConfLine() string {
  output := ""
  output += fmt.Sprintf("  server %s %s ", s.Name, s.Ip)
  output += fmt.Sprintf(" maxconn 32")
  if s.HealthCheck {
    output += " check"
  }
  if s.Backup {
    output += " backup"
  }
  output += "\n"
  return output
}
