package main

import (
  "flag"
  "github.com/garyburd/redigo/redis"
  "github.com/macb/go-digitalocean/digitalocean"
  "github.com/macb/go-haproxy/haproxy"
  "github.com/macb/go-elasticocean/elasticocean"
  "log"
  "os"
  "time"
)

// A few DigitalOcean constants.


var DO = new(digitalocean.DigitalOcean)
var deployCurrent bool
var web string
var E = new(elasticocean.Elastic)

func init() {
  start := time.Now()
  E.Servers = map[string]*elasticocean.Server{}
  flag.StringVar(&DO.ID, "id", "", "DigitalOcean ID")
  flag.StringVar(&DO.Key, "key", "", "DigitalOcean key")
  flag.StringVar(&web, "web", "", "Path to new web tar")
  flag.StringVar(&E.BaseName, "basename", "elastic-ocean", "Base name for all web slaves.")
  flag.StringVar(&E.ServerPort, "port", ":80", "Port for any web slaves.")
  flag.StringVar(&E.Haproxy.Config, "haconf", "/etc/haproxy/haproxy.cfg", "Path to HAProxy config file")

  E.Haproxy.Socket = haproxy.Socket(*flag.String("hasock", "/tmp/haproxy", "Path to haproxy socket file"))

  flag.BoolVar(&deployCurrent, "deploy", false, "Deploy current image?")

  redisServer := *flag.String("redis", ":6379", "Redis Server")

  flag.Parse()
  prepareRedis(redisServer)
  mustConfig()
  E.Initialize()
  load, _ := E.Haproxy.GetLoad(E.BaseName)
  for _, server := range load {
    if server.Name == "BACKEND" {
      continue
    }
    E.Servers[server.Name].Load = server
    log.Printf("Loaded: %s at %s", E.Servers[server.Name].Name, E.Servers[server.Name].Ip)
  }
  log.Print("Done initializing in: ", time.Now().Sub(start)*time.Millisecond)
}

func main() {
  if web != "" {
    // Deploy a new web image. Update redis with current image id
    if err := E.NewSnapshot(web); err != nil {
      panic(err)
    }
    log.Print("Snapshot created.")
    os.Exit(0)
  }

  if deployCurrent {
    var s *elasticocean.Server
    var err error

    if s, err = E.DeployCurrent(); err != nil {
      panic(err)
    }

    if err := E.AddSlave(s); err != nil {
      panic(err)
    }
    log.Print(s)
    os.Exit(0)
  }

  if err := E.Balance(); err != nil {
    log.Panic(err)
  }
}

func prepareRedis(server string) {
  E.RedisPool = redis.NewPool(func() (redis.Conn, error) {
    return redis.Dial("tcp", server)
  }, 10)
}

func mustConfig() {
  if DO.ID == "" || DO.Key == "" {
    panic("Must set DigitalOcean ID and Key")
  }
}
