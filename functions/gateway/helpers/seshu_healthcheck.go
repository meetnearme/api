package helpers

import (
	"runtime"
	"sync/atomic"
	"time"
)

var (
	seshuLoopAlive int32
	processStart   = time.Now()
)

// Called regularly by the loop to signal it is still alive
func MarkSeshuLoopAlive() {
	atomic.StoreInt32(&seshuLoopAlive, 1)
}

// Called by health check
func IsSeshuLoopAlive() bool {
	return atomic.LoadInt32(&seshuLoopAlive) == 1
}

func GetHealthStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"goroutines":   runtime.NumGoroutine(),
		"memory_alloc": m.Alloc / 1024,      // in KB
		"total_alloc":  m.TotalAlloc / 1024, // in KB
		"sys":          m.Sys / 1024,        // in KB
		"num_gc":       m.NumGC,
		"uptime_sec":   int(time.Since(processStart).Seconds()),
	}
}
