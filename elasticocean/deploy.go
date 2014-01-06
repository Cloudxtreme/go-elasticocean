package elasticocean

import (
  "errors"
  "fmt"
  "github.com/dchest/uniuri"
  "github.com/garyburd/redigo/redis"
  "github.com/macb/go-digitalocean/digitalocean"
  "os/exec"
  "path/filepath"
  "time"
)

func (e Elastic) DeployCurrent() (s *Server, err error) {
  var snapshotId int
  if snapshotId, err = e.getCurrentSnapshot(); err != nil {
    return nil, err
  }
  return e.deploySnapshot(snapshotId)
}

func (e Elastic) getCurrentSnapshot() (id int, err error) {
  c := e.RedisPool.Get()
  return redis.Int(c.Do("get", "elastic-ocean::current"))
}

func (e Elastic) deploySnapshot(id int) (s *Server, err error) {
  keys, err := e.DigitalOcean.SshKeys()
  if err != nil {
    return nil, err
  }

  uid := uniuri.NewLen(6)
  s = &Server{
    Name:    fmt.Sprintf("%s-%d-%s", e.BaseName, id, uid),
    ImageId: id,
    UID:     uid,
  }

  droplet, err := e.DigitalOcean.NewDroplet(s.Name, digitalocean.SIZE_512, id, digitalocean.NYC2_REGION, true, keys)
  if err != nil {
    return nil, err
  }

  if _, err := e.DigitalOcean.PollEvent(droplet.EventId); err != nil {
    return nil, err
  }

  droplet, err = e.DigitalOcean.DropletStatus(droplet.Id)
  s.Ip = droplet.IpAddress + e.ServerPort
  return s, err
}

func (e Elastic) NewSnapshot(path string) (err error) {
  // TODO Get ssh key ids
  keys, err := e.DigitalOcean.SshKeys()
  if err != nil {
    return err
  }

  droplet, err := e.DigitalOcean.NewDroplet("Elastic-Ocean-Temp-Snapshot", digitalocean.SIZE_512, digitalocean.UBUNTU_1310, digitalocean.NYC2_REGION, true, keys)
  if err != nil {
    return err
  }

  if _, err := e.DigitalOcean.PollEvent(droplet.EventId); err != nil {
    return err
  }

  // Get IP and what not
  if droplet, err = e.DigitalOcean.DropletStatus(droplet.Id); err != nil {
    return err
  }

  // FIXME: Sleep because digital-ocean returns progress of 100% but vm wasnt connectable
  time.Sleep(3 * time.Second)

  //Deploy given tgz on VM
  if err = deploy(path, droplet.IpAddress); err != nil {
    return err
  }

  // Shutdown before power off to avoid corruption.
  if _, err := e.DigitalOcean.PollEventErr(e.DigitalOcean.DropletShutdown(droplet.Id)); err != nil {
    return err
  }

  if _, err := e.DigitalOcean.PollEventErr(e.DigitalOcean.DropletPowerOff(droplet.Id)); err != nil {
    return err
  }

  //TODO Generate name based on something smart
  snapshotName := fmt.Sprint(time.Now().Unix())
  if _, err := e.DigitalOcean.PollEventErr(e.DigitalOcean.DropletSnapshot(droplet.Id, snapshotName)); err != nil {
    return err
  }

  if _, err := e.DigitalOcean.PollEventErr(e.DigitalOcean.DropletDestroy(droplet.Id)); err != nil {
    return err
  }

  image, err := e.DigitalOcean.FindImageByName(snapshotName)
  if err != nil {
    return err
  }

  if err := e.updateRedis(image); err != nil {
    return err
  }

  return nil
}

func deploy(path string, ip string) (err error) {
  scp := exec.Command("scp", "-o StrictHostKeyChecking=no", "-o UserKnownHostsFile=/dev/null", path, "root@"+ip+":~")
  if out, err := scp.Output(); err != nil {
    return errors.New(fmt.Sprintf("Error in scp of file at %s to %s - Out: %s", path, ip, out))
  }

  untar := exec.Command("ssh", "-o StrictHostKeyChecking=no", "-o UserKnownHostsFile=/dev/null", "root@"+ip, "tar -xf "+filepath.Base(path))
  if out, err := untar.Output(); err != nil {
    return errors.New(fmt.Sprintf("Error in untar at %s on %s - Out: %s", filepath.Base(path), ip, out))
  }

  // Removes extension from the base item ie .tgz/.tar/.zip/.tz
  untarDir := filepath.Base(path)[:len(filepath.Base(path))-len(filepath.Ext(path))]
  install := exec.Command("ssh", "-o StrictHostKeyChecking=no", "-o UserKnownHostsFile=/dev/null", "root@"+ip, "bash "+untarDir+"/install.sh")
  if out, err := install.Output(); err != nil {
    return errors.New(fmt.Sprintf("Error in install script at %s on %s - Out: %s", untarDir, ip, out))
  }
  return err
}

func (e Elastic) updateRedis(image *digitalocean.Image) error {
  c := e.RedisPool.Get()
  defer c.Close()
  args := redis.Args{}.Add("elastic-ocean::latest").Add(image.Id)
  _, err := c.Do("SET", args...)
  return err
}
