package client_conn

import (
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"math/rand"
	"sync"
	"time"
)

const Name = "kelvins-balancer"

func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(Name, &rrPickerBuilder{}, base.Config{HealthCheck: true})
}

func init() {
	balancer.Register(newBuilder())
}

type rrPickerBuilder struct{}

func (*rrPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	var scs []balancer.SubConn
	for sc := range info.ReadySCs {
		scs = append(scs, sc)
	}
	return &rrPicker{
		subConnes: scs,
		next:      randIntn(len(scs)),
	}
}

type rrPicker struct {
	subConnes []balancer.SubConn
	mu        sync.Mutex
	next      int
}

func (p *rrPicker) Pick(balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.Lock()
	sc := p.subConnes[p.next]
	p.next = (p.next + 1) % len(p.subConnes)
	p.mu.Unlock()
	return balancer.PickResult{SubConn: sc}, nil
}

var (
	r  = rand.New(rand.NewSource(time.Now().UnixNano()))
	mu sync.Mutex
)

func randInt() int {
	mu.Lock()
	defer mu.Unlock()
	return r.Int()
}

func randInt63n(n int64) int64 {
	mu.Lock()
	defer mu.Unlock()
	return r.Int63n(n)
}

func randIntn(n int) int {
	mu.Lock()
	defer mu.Unlock()
	return r.Intn(n)
}

func randFloat64() float64 {
	mu.Lock()
	defer mu.Unlock()
	return r.Float64()
}

func randUint64() uint64 {
	mu.Lock()
	defer mu.Unlock()
	return r.Uint64()
}
