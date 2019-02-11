package cmdutils

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/fatih/color"
)

func StartMemLogs() (onEnd func()) {
	globalStart := time.Now()
	allocatedMem := getAllocatedMemMB()
	allocatedMemMu := &sync.Mutex{}
	ticker := time.NewTicker(time.Second)

	log := func() {
		allocatedMemMu.Lock()
		allocatedMem = getAllocatedMemMB()
		allocatedMemMu.Unlock()
		timeSinceStartMin := int(time.Since(globalStart).Minutes())
		fmt.Fprintf(color.Output, "[%sm][%vMB] utilization\n", color.YellowString("%v", timeSinceStartMin), color.YellowString("%v", allocatedMem))
	}

	go func() {
		for {
			<-ticker.C
			log()
		}
	}()

	log()

	return func() {
		//ticker.Stop()
	}
}

func getAllocatedMemMB() int {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int(m.HeapAlloc / 1024 / 1024)
}
