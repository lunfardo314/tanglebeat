package bundle_source

import (
	"github.com/iotaledger/iota.go/trinary"
)

// the idea is abstract source (channel) of bundles to be confirmed.
// Channel produces records of type FirstBundleData to confirm.
// It can be newly created (e.g. new transfer), or it can be already read from the tangle.
// 'StartTime' and 'IsNew' is set accordingly
// 'address' is an input  address of the transfer (if it is a transfer)
//
// Traveling IOTA -style of BundleTrytes source produces sequences of transfer bundles, each
// spends whole balances from the previous address to the next
// It produces new budnles only upon and immediately after confirmation of the previous transfer.
//
// BundleTrytes source in principle it can be any BundleTrytes generator,
// for example sequence of MAM bundles to confirm

// structure produced by BundleTrytes generator
type FirstBundleData struct {
	Addr         trinary.Hash
	Index        uint64
	BundleTrytes []trinary.Trytes // raw bundle trytes to start with confirmation.
	IsNew        bool             // new BundleTrytes created or existing one found
	//StartTime             uint64           // unix milisec	set when bundle was read from tangle. Tx timestamps not used
	TotalDurationPoWMs    uint64 // > 0 if new BundleTrytes, ==0 if existing BundleTrytes
	TotalDurationTipselMs uint64 // > 0 if new BundleTrytes, ==0 if existing BundleTrytes
	NumAttach             uint64 // number of tails with the same BundleTrytes hash at the start
}

type BundleSourceChan chan *FirstBundleData