package test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBookPorts(t *testing.T) {
	ports := map[int]struct{}{}
	lock := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				p, _ := BookPorts(1, "localhost")
				lock.Lock()
				_, ok := ports[p[0]]
				assert.False(t, ok)
				ports[p[0]] = struct{}{}
				lock.Unlock()
			}
			wg.Done()
		}()
	}
	wg.Wait()
	fmt.Println("done")
}
