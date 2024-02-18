package status

import (
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	Inactive = iota
	Active
)

var (
	active int32
	wg     sync.WaitGroup
)

func init() {
	active = 1
}

func IsRunning() bool {
	return atomic.LoadInt32(&active) == Active
}

func Shutdown() {
	atomic.StoreInt32(&active, Inactive)
	fmt.Println("is run false")
}

func AddWaitGroup() {
	wg.Add(1)
}

func DoneWaitGroup() {
	wg.Done()
}

func WaitGroup() {
	wg.Wait()
}
