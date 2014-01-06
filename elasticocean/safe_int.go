package elasticocean

import (
  "sync/atomic"
)

type safeInt struct {
  value int64
}

func (s *safeInt) Incr() {
  atomic.AddInt64(&s.value, 1)
}

func (s *safeInt) Decr() {
  atomic.AddInt64(&s.value, -1)
}

func (s *safeInt) Value() int64 {
  return s.value
}
