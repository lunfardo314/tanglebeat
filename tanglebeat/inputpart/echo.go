package inputpart

import (
	"github.com/unioproject/tanglebeat/lib/utils"
	"github.com/unioproject/tanglebeat/tanglebeat/hashcache"
	"time"
)

const (
	echoBufferHashLen            = 12
	echoBufferSegmentDurationSec = 60
	echoBufferRetentionPeriodSec = 30 * 60
)

type echoEntry struct {
	whenSent      uint64
	seen          bool
	whenSeenFirst uint64
	whenSeenLast  uint64
}

var (
	echoBuffer *hashcache.HashCacheBase
)

func startEchoLatencyRoutine() {
	echoBuffer = hashcache.NewHashCacheBase(
		"echoBuffer", echoBufferHashLen, echoBufferSegmentDurationSec, echoBufferRetentionPeriodSec)
	go func() {
		debugf("Started echo latency calculation routine")
		var percNotSeen, avgSeenFirstMs, avgSeenLastMs uint64
		for {
			time.Sleep(10 * time.Second)
			percNotSeen, avgSeenFirstMs, avgSeenLastMs = calcAvgEchoParams()
			updateEchoMetrics(percNotSeen, avgSeenFirstMs, avgSeenLastMs)
		}
	}()
}

func TxSentForEcho(txhash string, ts uint64) {
	var entry hashcache.CacheEntry

	ee := echoEntry{
		whenSent: ts,
	}
	if txcache.FindNoTouch(txhash, &entry) {
		ee.whenSeenFirst = entry.FirstSeen
		ee.whenSeenLast = entry.LastSeen
		ee.seen = true
	}
	echoBuffer.SeenHashBy(txhash, 0, &ee, nil)
	debugf("++++++Promo tx waiting for echo: %v...", txhash[:12])
}

// it is called for each tx message
func checkForEcho(txhash string, ts uint64) {
	var entry hashcache.CacheEntry
	echoBuffer.Lock()
	defer echoBuffer.Unlock()

	if echoBuffer.FindNoTouch__(txhash, &entry) {
		d := entry.Data.(*echoEntry)
		if d.seen {
			debugf("+++++++ Promo tx echo in %v msec. %v..", ts-d.whenSent, txhash[:12])
			d.whenSeenLast = ts
		} else {
			d.whenSeenFirst = ts
			d.whenSeenLast = ts
			d.seen = true
		}
	}
}

func nonNegativeDuration(whenSent, whenSeenFirst uint64) uint64 {
	var ret int64
	ret = int64(whenSeenFirst) - int64(whenSent)
	if ret < 0 {
		ret = 0
	}
	return uint64(ret)
}

func calcAvgEchoParams() (uint64, uint64, uint64) {
	var numAll, numSeen, avgSeenFirstLatencyMs, avgSeenLastLatencyMs uint64
	var data *echoEntry
	earliest := utils.UnixMsNow() - 30*60*1000 //30min
	echoBuffer.ForEachEntry(func(entry *hashcache.CacheEntry) {
		data = entry.Data.(*echoEntry)
		numAll++
		if data.seen {
			numSeen++
			avgSeenFirstLatencyMs += nonNegativeDuration(data.whenSent, data.whenSeenFirst)
			avgSeenLastLatencyMs += nonNegativeDuration(data.whenSent, data.whenSeenLast)
		}
	}, earliest, true)
	var percNotSeen uint64
	// averages are calculated only if enough data
	if numSeen > 5 {
		avgSeenFirstLatencyMs = avgSeenFirstLatencyMs / numSeen
		avgSeenLastLatencyMs = avgSeenLastLatencyMs / numSeen
		percNotSeen = 100 - (numSeen*100)/numAll
	} else {
		avgSeenFirstLatencyMs = 0
		avgSeenLastLatencyMs = 0
		percNotSeen = 0
	}
	debugf("percNotSeen = %v avgSeenFirstLatencyMs = %v avgSeenLastLatencyMs = %v",
		percNotSeen, avgSeenFirstLatencyMs, avgSeenLastLatencyMs)

	if avgSeenFirstLatencyMs > 100000 {
		debugf("Anomaly avgSeenFirstLatencyMs = %v", avgSeenFirstLatencyMs)
	}
	return percNotSeen, avgSeenFirstLatencyMs, avgSeenLastLatencyMs
}
