package loop

import (
	"log"
	"opensource/lib/constant"
	"opensource/lib/status"
	"sync"
	"time"
)

var srv = &Service{
	FuncMap: make(map[string]*FuncState),
}

type Service struct {
	FuncMap map[string]*FuncState
	Mu      sync.Mutex
}

type Func func(name string, options interface{}, closed chan struct{}, cf func(input interface{}))

type FuncState struct {
	name    string
	call    Func
	state   int
	options interface{}
	closed  chan struct{}
	cf      func(input interface{})
}

func NewLoopFunc(name string, fb Func, options interface{}, cf func(input interface{})) {
	f := &FuncState{
		name:    name,
		call:    fb,
		state:   constant.StatusInactive,
		options: options,
		closed:  make(chan struct{}),
		cf:      cf,
	}
	srv.Mu.Lock()
	defer srv.Mu.Unlock()
	srv.FuncMap[f.name] = f
}

func Run() {
	go loop()
}

func loop() {
	retry()
	for status.IsRunning() {
		select {
		case <-time.After(5 * time.Second):
			retry()
		}
	}
}

func retry() {
	for key, f := range srv.FuncMap {
		if f.state != constant.StatusActive {
			log.Printf("name:%s SetState", f.name)
			SetState(key, constant.StatusActive)
			go f.call(f.name, f.options, f.closed, f.cf)
		}
	}
}

func SetState(key string, state int) {
	srv.Mu.Lock()
	defer srv.Mu.Unlock()
	if status.IsRunning() {
		srv.FuncMap[key].state = state
	}
}

func Stop() {
	for _, f := range srv.FuncMap {
		if f.state == constant.StatusActive {
			f.closed <- struct{}{}
		}
	}
	log.Println("loop func stop")
}
