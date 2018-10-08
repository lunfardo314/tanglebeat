package main

import (
	"github.com/lunfardo314/giota"
	"github.com/lunfardo314/tanglebeat/comm"
	"github.com/lunfardo314/tanglebeat/confirmer"
	"github.com/lunfardo314/tanglebeat/lib"
	"github.com/op/go-logging"
	"time"
)

func (seq *Sequence) startConfirmer(bundle giota.Bundle, log *logging.Logger) chan *confirmer.ConfirmerUpdate {
	ret := confirmer.Confirmer{
		IOTANode:              seq.Params.IOTANode[0],
		IOTANodeGTTA:          seq.Params.IOTANodeGTTA[0],
		IOTANodeATT:           seq.Params.IOTANodeATT[0],
		TimeoutAPI:            seq.Params.TimeoutAPI,
		TimeoutGTTA:           seq.Params.TimeoutGTTA,
		TimeoutATT:            seq.Params.TimeoutATT,
		TxTagPromote:          seq.TxTagPromote,
		ForceReattachAfterMin: seq.Params.ForceReattachAfterMin,
		PromoteNoChain:        seq.Params.PromoteNoChain,
		PromoteEverySec:       seq.Params.PromoteEverySec,
	}
	return ret.Run(bundle, log)
}

func (seq *Sequence) publishSenderUpdate(updConf *confirmer.ConfirmerUpdate, addr giota.Address, index int, sendingStarted time.Time) {
	upd := comm.SenderUpdate{
		SeqUID:                seq.UID,
		SeqName:               seq.Name,
		UpdType:               updConf.UpdateType,
		Index:                 index,
		Addr:                  addr,
		SendingStartedTs:      lib.UnixMs(sendingStarted),
		NumAttaches:           updConf.Stats.NumAttaches,
		NumPromotions:         updConf.Stats.NumPromotions,
		NodeATT:               seq.Params.IOTANodeATT[0],
		NodeGTTA:              seq.Params.IOTANodeGTTA[0],
		PromoteEveryNumSec:    seq.Params.PromoteEverySec,
		ForceReattachAfterSec: seq.Params.ForceReattachAfterMin,
		PromoteNochain:        seq.Params.PromoteNoChain,
	}
	timeSinceStart := time.Since(sendingStarted)
	timeSinceStartMsec := int64(timeSinceStart / time.Millisecond)
	upd.SinceSendingMsec = timeSinceStartMsec
	securityLevel := 2
	upd.BundleSize = securityLevel + 2
	upd.PromoBundleSize = 1
	totalTx := upd.BundleSize*upd.NumAttaches + upd.PromoBundleSize*upd.NumPromotions
	if updConf.Stats.NumATT != 0 && totalTx != 0 {
		upd.AvgPoWDurationPerTxMsec = updConf.Stats.TotalDurationATTMsec / int64(updConf.Stats.NumATT*totalTx)
	}
	if updConf.Stats.NumGTTA != 0 {
		upd.AvgGTTADurationMsec = updConf.Stats.TotalDurationGTTAMsec / int64(updConf.Stats.NumGTTA)
	}
	timeSinceStartSec := float32(timeSinceStartMsec) / float32(1000)
	if timeSinceStartSec > 0.1 {
		upd.TPS = float32(totalTx) / timeSinceStartSec
	} else {
		upd.TPS = 0
	}
	publishUpdate(&upd)
}
