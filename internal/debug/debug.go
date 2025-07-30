package debug

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/gorilla/mux"
)

// DebugServer 调试服务器
type DebugServer struct {
	server *http.Server
	router *mux.Router
}

// NewDebugServer 创建调试服务器
func NewDebugServer(port int) *DebugServer {
	router := mux.NewRouter()
	
	ds := &DebugServer{
		router: router,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      router,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
	}
	
	ds.setupRoutes()
	return ds
}

// setupRoutes 设置路由
func (ds *DebugServer) setupRoutes() {
	// pprof性能分析端点
	ds.router.HandleFunc("/debug/pprof/", pprof.Index)
	ds.router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	ds.router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	ds.router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	ds.router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	ds.router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	ds.router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	ds.router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	ds.router.Handle("/debug/pprof/block", pprof.Handler("block"))
	ds.router.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	ds.router.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	
	// 自定义调试端点
	ds.router.HandleFunc("/debug/runtime", ds.runtimeHandler)
	ds.router.HandleFunc("/debug/gc", ds.gcHandler)
	ds.router.HandleFunc("/debug/build", ds.buildHandler)
	ds.router.HandleFunc("/debug/vars", ds.varsHandler)
	ds.router.HandleFunc("/debug/stack", ds.stackHandler)
}

// Start 启动调试服务器
func (ds *DebugServer) Start() error {
	return ds.server.ListenAndServe()
}

// Stop 停止调试服务器
func (ds *DebugServer) Stop() error {
	return ds.server.Close()
}

// runtimeHandler 运行时信息处理器
func (ds *DebugServer) runtimeHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	info := map[string]interface{}{
		"timestamp":        time.Now().Format(time.RFC3339),
		"go_version":       runtime.Version(),
		"goos":            runtime.GOOS,
		"goarch":          runtime.GOARCH,
		"num_cpu":         runtime.NumCPU(),
		"num_goroutine":   runtime.NumGoroutine(),
		"num_cgo_call":    runtime.NumCgoCall(),
		"compiler":        runtime.Compiler,
		"memory": map[string]interface{}{
			"alloc":           m.Alloc,
			"total_alloc":     m.TotalAlloc,
			"sys":            m.Sys,
			"lookups":        m.Lookups,
			"mallocs":        m.Mallocs,
			"frees":          m.Frees,
			"heap_alloc":     m.HeapAlloc,
			"heap_sys":       m.HeapSys,
			"heap_idle":      m.HeapIdle,
			"heap_inuse":     m.HeapInuse,
			"heap_released":  m.HeapReleased,
			"heap_objects":   m.HeapObjects,
			"stack_inuse":    m.StackInuse,
			"stack_sys":      m.StackSys,
			"gc_sys":         m.GCSys,
			"other_sys":      m.OtherSys,
		},
		"gc": map[string]interface{}{
			"next_gc":        m.NextGC,
			"last_gc":        time.Unix(0, int64(m.LastGC)).Format(time.RFC3339),
			"num_gc":         m.NumGC,
			"num_forced_gc":  m.NumForcedGC,
			"pause_total_ns": m.PauseTotalNs,
			"pause_ns":       m.PauseNs,
			"gc_cpu_fraction": m.GCCPUFraction,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// gcHandler GC信息处理器
func (ds *DebugServer) gcHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// 触发GC
		runtime.GC()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("GC triggered"))
		return
	}
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	gcInfo := map[string]interface{}{
		"timestamp":       time.Now().Format(time.RFC3339),
		"num_gc":          m.NumGC,
		"num_forced_gc":   m.NumForcedGC,
		"pause_total_ns":  m.PauseTotalNs,
		"last_gc":         time.Unix(0, int64(m.LastGC)).Format(time.RFC3339),
		"next_gc":         m.NextGC,
		"gc_cpu_fraction": m.GCCPUFraction,
		"enable_gc":       m.EnableGC,
		"debug_gc":        m.DebugGC,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gcInfo)
}

// buildHandler 构建信息处理器
func (ds *DebugServer) buildHandler(w http.ResponseWriter, r *http.Request) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		http.Error(w, "Build info not available", http.StatusNotFound)
		return
	}
	
	info := map[string]interface{}{
		"go_version": buildInfo.GoVersion,
		"path":       buildInfo.Path,
		"main":       buildInfo.Main,
		"deps":       buildInfo.Deps,
		"settings":   buildInfo.Settings,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// varsHandler 环境变量处理器
func (ds *DebugServer) varsHandler(w http.ResponseWriter, r *http.Request) {
	vars := make(map[string]interface{})
	
	// 运行时统计
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	vars["runtime"] = map[string]interface{}{
		"goroutines": runtime.NumGoroutine(),
		"memory_mb":  m.Alloc / 1024 / 1024,
		"gc_runs":    m.NumGC,
	}
	
	// 系统信息
	vars["system"] = map[string]interface{}{
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"cpus":     runtime.NumCPU(),
		"version":  runtime.Version(),
	}
	
	vars["timestamp"] = time.Now().Format(time.RFC3339)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vars)
}

// stackHandler 堆栈跟踪处理器
func (ds *DebugServer) stackHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	
	// 获取所有goroutine的堆栈信息
	buf := make([]byte, 1024*1024) // 1MB buffer
	n := runtime.Stack(buf, true)
	w.Write(buf[:n])
}

// PerformanceProfiler 性能分析器
type PerformanceProfiler struct {
	startTime time.Time
	samples   []ProfileSample
}

// ProfileSample 性能样本
type ProfileSample struct {
	Timestamp  time.Time     `json:"timestamp"`
	Operation  string        `json:"operation"`
	Duration   time.Duration `json:"duration"`
	MemBefore  uint64        `json:"mem_before"`
	MemAfter   uint64        `json:"mem_after"`
	Goroutines int           `json:"goroutines"`
}

// NewPerformanceProfiler 创建性能分析器
func NewPerformanceProfiler() *PerformanceProfiler {
	return &PerformanceProfiler{
		startTime: time.Now(),
		samples:   make([]ProfileSample, 0, 1000),
	}
}

// Profile 分析性能
func (pp *PerformanceProfiler) Profile(operation string, fn func()) {
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	goroutinesBefore := runtime.NumGoroutine()
	
	start := time.Now()
	fn()
	duration := time.Since(start)
	
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	
	sample := ProfileSample{
		Timestamp:  start,
		Operation:  operation,
		Duration:   duration,
		MemBefore:  memBefore.Alloc,
		MemAfter:   memAfter.Alloc,
		Goroutines: goroutinesBefore,
	}
	
	pp.samples = append(pp.samples, sample)
	
	// 保持最近1000个样本
	if len(pp.samples) > 1000 {
		pp.samples = pp.samples[1:]
	}
}

// GetSamples 获取性能样本
func (pp *PerformanceProfiler) GetSamples() []ProfileSample {
	return pp.samples
}

// GetStats 获取统计信息
func (pp *PerformanceProfiler) GetStats() map[string]interface{} {
	if len(pp.samples) == 0 {
		return map[string]interface{}{
			"sample_count": 0,
			"uptime":       time.Since(pp.startTime),
		}
	}
	
	var totalDuration time.Duration
	var maxDuration time.Duration
	var minDuration = pp.samples[0].Duration
	operationCounts := make(map[string]int)
	
	for _, sample := range pp.samples {
		totalDuration += sample.Duration
		if sample.Duration > maxDuration {
			maxDuration = sample.Duration
		}
		if sample.Duration < minDuration {
			minDuration = sample.Duration
		}
		operationCounts[sample.Operation]++
	}
	
	avgDuration := totalDuration / time.Duration(len(pp.samples))
	
	return map[string]interface{}{
		"sample_count":      len(pp.samples),
		"uptime":           time.Since(pp.startTime),
		"avg_duration":     avgDuration,
		"max_duration":     maxDuration,
		"min_duration":     minDuration,
		"total_duration":   totalDuration,
		"operation_counts": operationCounts,
	}
}