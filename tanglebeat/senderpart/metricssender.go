package senderpart

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unioproject/tanglebeat/tbsender/sender_update"
)

var (
	confCounter                  *prometheus.CounterVec
	confPoWCostCounter           *prometheus.CounterVec
	confDurationSecCounter       *prometheus.CounterVec
	confPoWDurationSecCounter    *prometheus.CounterVec
	confTipselDurationSecCounter *prometheus.CounterVec
	//restartCounter               prometheus.Counter
)

func init() {
	confCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tanglebeat_confirmation_counter",
		Help: "Increases every time sender confirms a transfer",
	}, []string{"seqid"})

	confPoWCostCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tanglebeat_pow_cost_counter",
		Help: "Counter for number of tx attached during the confirmation = num. attachments * Bundle size + num. promotions * promo Bundle size",
	}, []string{"seqid"})

	confDurationSecCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tanglebeat_confirmation_duration_counter",
		Help: "Sums up confirmation durations of the transfer.",
	}, []string{"seqid"})

	confPoWDurationSecCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tanglebeat_pow_duration_counter",
		Help: "Sums up total duration it took to do PoW for confirmation.",
	}, []string{"seqid", "node_pow"})

	confTipselDurationSecCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tanglebeat_tipsel_duration_counter",
		Help: "Sums up total duration it took to do tip selection for confirmation.",
	}, []string{"seqid", "node_tipsel"})

	//restartCounter = prometheus.NewCounter(prometheus.CounterOpts{
	//	Name: "tanglebeat_restart_counter",
	//	Help: "Increases every time program starts",
	//})
	prometheus.MustRegister(confCounter)
	prometheus.MustRegister(confDurationSecCounter)
	prometheus.MustRegister(confPoWCostCounter)
	prometheus.MustRegister(confPoWDurationSecCounter)
	prometheus.MustRegister(confTipselDurationSecCounter)
	//prometheus.MustRegister(restartCounter)
}

//var restarted = time.Now()
//var restartIncreased = false

func updateSenderMetrics(upd *sender_update.SenderUpdate) {
	//if !restartIncreased {
	//	if time.Since(restarted) > 30*time.Second {
	//		// increase restart counter only 30 sec after restart
	//		// to give  time to prometheus to scrape 0 Balance
	//		restartCounter.Inc()
	//		restartIncreased = true
	//	}
	//}
	if upd.UpdType != sender_update.SENDER_UPD_CONFIRM {
		return
	}
	debugf("'confirm' received. Update metrics for %v(%v), Index = %v",
		upd.SeqUID, upd.SeqName, upd.Index)

	confCounter.With(prometheus.Labels{"seqid": upd.SeqUID}).Inc()

	durSec := float64(upd.UpdateTs-upd.StartTs) / 1000
	confDurationSecCounter.
		With(prometheus.Labels{"seqid": upd.SeqUID}).Add(durSec)

	powCost := float64(upd.NumAttaches*upd.BundleSize + upd.NumPromotions*upd.PromoBundleSize)
	confPoWCostCounter.
		With(prometheus.Labels{"seqid": upd.SeqUID}).Add(powCost)

	confPoWDurationSecCounter.
		With(prometheus.Labels{
			"seqid":    upd.SeqUID,
			"node_pow": upd.NodePOW,
		}).Add(float64(upd.TotalPoWMsec) / 1000)

	confTipselDurationSecCounter.
		With(prometheus.Labels{
			"seqid":       upd.SeqUID,
			"node_tipsel": upd.NodeTipsel,
		}).Add(float64(upd.TotalTipselMsec) / 1000)
}
