package main

import (
	"encoding/json"
	"fmt"
	"github.com/unioproject/tanglebeat/lib/utils"
	"github.com/unioproject/tanglebeat/tanglebeat/cfg"
	"github.com/unioproject/tanglebeat/tanglebeat/inputpart"
	"math"
	"net"
	"net/url"
	"runtime"
	"sync"
	"time"
)

type GlbStats struct {
	InstanceVersion     string                         `json:"instanceVersion"`
	InstanceStarted     uint64                         `json:"instanceStarted"`
	QuorumTX            int                            `json:"quorumTX"`
	QuorumSN            int                            `json:"quorumSN"`
	QuorumLMI           int                            `json:"quorumLMI"`
	GoRuntimeStats      memStatsStruct                 `json:"goRuntimeStats"`
	ZmqCacheStats       inputpart.ZmqCacheStatsStruct  `json:"zmqRuntimeStats"`
	ZmqOutputStats      inputpart.ZmqOutputStatsStruct `json:"zmqOutputStats"`
	ZmqOutputStats10min inputpart.ZmqOutputStatsStruct `json:"zmqOutputStats10min"`
	ZmqInputStats       []*inputpart.ZmqRoutineStats   `json:"zmqInputStats"`

	mutex *sync.RWMutex
}

type memStatsStruct struct {
	MemAllocMB   float64 `json:"memAllocMB"`
	NumGoroutine int     `json:"numGoroutine"`
	mutex        *sync.Mutex
}

var glbStats = &GlbStats{
	InstanceVersion: cfg.Version,
	InstanceStarted: utils.UnixMsNow(),
	mutex:           &sync.RWMutex{},
}

func initGlobStatsCollector(refreshEverySec int) {

	inputpart.InitZmqStatsCollector(refreshEverySec)
	go updateGlbStatsLoop(refreshEverySec)
}

func updateGlbStatsLoop(refreshStatsEverySec int) {
	var mem runtime.MemStats
	for {
		runtime.ReadMemStats(&mem)

		inp := inputpart.GetInputStats()

		glbStats.mutex.Lock()

		glbStats.ZmqInputStats = inp
		glbStats.ZmqCacheStats = *inputpart.GetZmqCacheStats()
		t1, t2 := inputpart.GetOutputStats()
		glbStats.ZmqOutputStats, glbStats.ZmqOutputStats10min = *t1, *t2

		glbStats.GoRuntimeStats.MemAllocMB = math.Round(100*(float64(mem.Alloc/1024)/1024)) / 100
		updateRuntimeMetrics(glbStats.GoRuntimeStats.MemAllocMB)

		glbStats.GoRuntimeStats.NumGoroutine = runtime.NumGoroutine()

		glbStats.QuorumTX = inputpart.GetTxQuorum()
		glbStats.QuorumSN = inputpart.GetSnQuorum()
		glbStats.QuorumLMI = inputpart.GetLmiQuorum()

		glbStats.mutex.Unlock()

		time.Sleep(time.Duration(refreshStatsEverySec) * time.Second)
	}
}

func getGlbStatsJSON(formatted bool, maskIP bool, hideInactive bool) []byte {
	glbStats.mutex.RLock()
	defer glbStats.mutex.RUnlock()

	toMarshal := getMaskedGlbStats(maskIP, hideInactive)
	var data []byte
	var err error
	if formatted {
		data, err = json.MarshalIndent(toMarshal, "", "   ")
	} else {
		data, err = json.Marshal(toMarshal)
	}
	if err != nil {
		return []byte(fmt.Sprintf("marshal error: %v", err))
	}
	return data
}

func isActiveRoutine(r *inputpart.ZmqRoutineStats) bool {
	if !r.Running {
		return false
	}
	if utils.SinceUnixMs(r.LastHeartbeatTs) > 10*10*1000 {
		return false
	}
	return true
}

func isIpAddr(uri string) bool {
	p, err := url.Parse(uri)
	if err != nil {
		return false
	}
	return net.ParseIP(p.Hostname()) != nil
}

func getMaskedGlbStats(maskIP bool, hideInactive bool) *GlbStats {
	if !maskIP && !hideInactive {
		return glbStats
	}

	maskedInputs := make([]*inputpart.ZmqRoutineStats, 0, len(glbStats.ZmqInputStats))
	for _, inp := range glbStats.ZmqInputStats {
		if !hideInactive || isActiveRoutine(inp) {
			if maskIP && isIpAddr(inp.Uri) {
				tmp := *inp
				tmp.Uri = "IP addr (masked)"
				maskedInputs = append(maskedInputs, &tmp)
			} else {
				maskedInputs = append(maskedInputs, inp)
			}
		}
	}
	ret := *glbStats
	ret.ZmqInputStats = maskedInputs
	return &ret
}
