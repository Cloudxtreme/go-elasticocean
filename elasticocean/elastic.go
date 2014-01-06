package elasticocean

import (
  "github.com/garyburd/redigo/redis"
  "github.com/macb/go-digitalocean/digitalocean"
  "github.com/macb/go-haproxy/haproxy"
  "sync"
  "time"
)

type Elastic struct {
  DigitalOcean digitalocean.DigitalOcean
  Haproxy      haproxy.Haproxy
  ServerPort   string
  BaseName     string
  RedisPool    *redis.Pool
  Servers      map[string]*Server
  Deploying    safeInt
  ServerLock   sync.Mutex
  FileLock     sync.Mutex
}

func (e Elastic) Initialize() (err error) {
  return e.loadhaproxyConf()
}

func (e Elastic) Balance() (err error) {
  t := time.NewTicker(1 * time.Second)
  for _ = range t.C {
    if err = e.checkLoad(); err != nil {
      return err
    }
  }
  return nil
}
