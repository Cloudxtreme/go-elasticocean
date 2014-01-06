package elasticocean

import (
  "github.com/macb/go-haproxy/haproxy"
  "log"
)

func (e *Elastic) checkLoad() error {
  var loadScore int64
  // We don't want to have more servers deploying than we have servers out.
  // Just seems silly.
  // TODO Configurable.

  e.ServerLock.Lock()
  serverCount := int64(len(e.Servers))
  e.ServerLock.Unlock()

  if e.Deploying.Value() >= serverCount {
    log.Print("Too many servers deploying, skipping load check.")
    return nil
  }

  load, err := e.Haproxy.GetLoad(e.BaseName)
  if err != nil {
    return err
  }
  for _, serverLoad := range load {
    if serverLoad.Name == "BACKEND" {
      continue
    }
    server := e.Servers[serverLoad.Name]
    overloaded := serverOverloaded(server.Load, serverLoad)
    switch server.Backup {
    case true:
      if !overloaded {
        server.Backup = false
      }
    case false:
      if overloaded {
        // A server is overloaded. Let's pick up the slack.
        loadScore++
      }
    }
    // Replace previous load with the new one.
    e.Servers[serverLoad.Name].Load = serverLoad
  }

  serversNeeded := loadScore - e.Deploying.Value()
  if serversNeeded > 0 {
    var i int64
    for i = 0; i < serversNeeded; i++ {
      go e.newSlave()
    }
  }
  return nil
}

func serverOverloaded(previous *haproxy.Load, current *haproxy.Load) bool {
  // TODO This could call funcs that match an interface.
  // That way a user can define more overloaded criteria.
  if current.Health != "UP" && current.Health != "INI" {
    return true
  }

  if current.FailedCheck > previous.FailedCheck {
    return true
  }

  return false
}

func (e *Elastic) newSlave() (*Server, error) {
  e.Deploying.Incr()
  log.Print("Deploying a new slave.")
  s, err := e.DeployCurrent()
  if err != nil {
    return s, err
  }
  if err = e.AddSlave(s); err != nil {
    return s, err
  }
  e.Deploying.Decr()
  return s, nil
}
