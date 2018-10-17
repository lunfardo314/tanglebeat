package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/lunfardo314/tanglebeat/lib"
	"github.com/lunfardo314/tanglebeat/pubsub"
	_ "github.com/mattn/go-sqlite3"
	"path"
	"strings"
	"sync"
	"time"
)

var dbconn *sql.DB
var dbCache1h map[pkey]*transferRecordWOPK
var dbmutext sync.Mutex

func initDB() {
	log.Infof("Database file: '%v'", Config.DbFile)
	var err error
	dbPathName := path.Join(Config.SiteDataDir, Config.DbFile)
	dbconn, err = sql.Open("sqlite3", dbPathName+"?_timeout=5000")
	if err != nil {
		log.Panicf("Failed to open dbconn at %v: %v%", dbPathName, err)
	}
	createTables()
}

func createTables() {
	tx, err := dbconn.Begin()
	if err != nil {
		log.Panicf("Failed to Begin dbconn tx %v%", err)
	}

	sqlTableTempl :=
		`CREATE TABLE IF NOT EXISTS transfers (
			seqid char(%d) not null,
			idx integer not null,
	        addr char(81) not null unique,
			seqname char(40),
			lastState char(10) not null,
			started_ts_msec integer,
			last_update_msec integer,
			num_attaches integer,
			num_promotions integer,
			total_pow_duration_msec integer,
			total_tipsel_duration_msec integer,
			node_att char(50),
			node_gtta char(50),
			bundle_size integer,
			promo_bundle_size integer,
			promote_every_sec integer,
			force_reattach_every_min integer,
			promote_chain_yn char(1),
            primary key (seqid, idx)
		)`
	sqlTable := fmt.Sprintf(sqlTableTempl, pubsub.SEQUENCE_UID_LEN)
	_, err = dbconn.Exec(sqlTable)
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	if err != nil {
		log.Panicf("Failed to create table dbconn tx %v\nSQL = %v", err, sqlTable)
	}
}

var allColumns = []string{
	"seqid", "idx", "addr", "seqname", "lastState", "started_ts_msec", "last_update_msec",
	"num_attaches", "num_promotions", "total_pow_duration_msec", "total_tipsel_duration_msec",
	"node_att", "node_gtta", "bundle_size", "promo_bundle_size", "promote_every_sec",
	"force_reattach_every_min", "promote_chain_yn",
}

type transferRecordWOPK struct {
	addr                       string
	seqname                    string
	lastState                  string
	started_ts_msec            int64
	last_update_msec           int64
	num_attaches               int64
	num_promotions             int64
	total_pow_duration_msec    int64
	total_tipsel_duration_msec int64
	node_att                   string
	node_gtta                  string
	bundle_size                int64
	promo_bundle_size          int64
	promote_every_sec          int64
	force_reattach_every_min   int64
	promote_chain_yn           string
}

func transferRecWOPKFromUpdate(upd *pubsub.SenderUpdate) *transferRecordWOPK {
	return &transferRecordWOPK{
		addr:                       string(upd.Addr),
		seqname:                    upd.SeqName,
		lastState:                  string(upd.UpdType),
		started_ts_msec:            upd.SendingStartedTs,
		last_update_msec:           upd.UpdateTs,
		num_attaches:               int64(upd.NumAttaches),
		num_promotions:             int64(upd.NumPromotions),
		total_pow_duration_msec:    upd.TotalPoWMsec,
		total_tipsel_duration_msec: upd.TotalTipselMsec,
		node_att:                   upd.NodeATT,
		node_gtta:                  upd.NodeGTTA,
		bundle_size:                int64(upd.BundleSize),
		promo_bundle_size:          int64(upd.PromoBundleSize),
		promote_every_sec:          int64(upd.PromoteEveryNumSec),
		force_reattach_every_min:   int64(upd.ForceReattachAfterMin),
		promote_chain_yn:           lib.Iff(upd.PromoteChain, "Y", "N").(string),
	}
}

type pkey struct {
	seqid string
	idx   int64
}

func runUpdateDb() {
	chanUpdate, err := pubsub.OpenSenderUpdateChan(Config.SenderURI, log)
	if err != nil {
		log.Criticalf("can't get new sub socket: %v", err)
	}

	log.Info("Started listening to data stream from sender")
	var pk pkey
	for upd := range chanUpdate {
		if upd.UpdateTs >= lib.UnixMs(time.Now())-60*60*1000 {
			// just in case filter older updates
			pk.seqid = upd.SeqUID
			pk.idx = int64(upd.Index)
			prec := transferRecWOPKFromUpdate(upd)
			// TODO deal with started_ts_msec
			err = writeDbAndCache(&pk, prec)
			if err != nil {
				log.Errorf("Error from updateRecord: %v", err)
			} else {
				log.Infof("Update '%v' seq = %v(%v), index = %v", upd.UpdType, upd.SeqUID, upd.SeqName, upd.Index)
			}
			ensureLast1h()
		}
	}
}

func writeDbAndCache(pk *pkey, prec *transferRecordWOPK) error {
	dbmutext.Lock()
	defer dbmutext.Unlock()

	err := writeRecordToDB(pk, prec)
	if err == nil {
		dbCache1h[*pk] = prec
	}
	return err
}

var sqlSelect1h = "select * from transfers where last_update_msec >= ?"

func read1hFromDB() error {
	dbCache1h = make(map[pkey]*transferRecordWOPK)
	rows, err := dbconn.Query(sqlSelect1h, lib.UnixMs(time.Now())-60*60*1000)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ptr *transferRecordWOPK
	var pk pkey

	for rows.Next() {
		ptr = &transferRecordWOPK{}
		err = rows.Scan(&pk.seqid, &pk.idx,
			&ptr.addr, &ptr.seqname, &ptr.lastState, &ptr.started_ts_msec, &ptr.last_update_msec,
			&ptr.num_attaches, &ptr.num_promotions,
			&ptr.total_pow_duration_msec, &ptr.total_tipsel_duration_msec,
			&ptr.node_att, &ptr.node_gtta, &ptr.bundle_size, &ptr.promo_bundle_size, &ptr.promote_every_sec,
			&ptr.force_reattach_every_min, &ptr.promote_chain_yn)
		if err != nil {
			return err
		}
		dbCache1h[pk] = ptr
	}
	return nil
}

func ensureLast1h() {
	oldest := lib.UnixMs(time.Now()) - 60*60*1000
	for k, v := range dbCache1h {
		if v.last_update_msec < oldest {
			delete(dbCache1h, k)
		}
	}
}

func existsRecInDb(pk *pkey) (bool, error) {
	sqlSelect := "select idx from transfers where seqid=? and idx=?"
	var idx int64
	row := dbconn.QueryRow(sqlSelect, pk.seqid, pk.idx)
	switch err := row.Scan(&idx); err {
	case sql.ErrNoRows:
		return false, nil
	case nil:
		return true, nil
	default:
		return false, err
	}
}

const insertTemplate = "insert into transfers (%s) values (%s)"
const updateTemplate = "update transfers set %s where %s"

func makeInsertSql(cols []string) string {
	qmlist := make([]string, len(cols))
	for i := range qmlist {
		qmlist[i] = "?"
	}
	return fmt.Sprintf(insertTemplate, strings.Join(allColumns, ", "), strings.Join(qmlist, ", "))
}

func makeUpdateSql(updateCols []string, pkCols []string) string {
	var colAssignList []string
	for _, col := range updateCols {
		colAssignList = append(colAssignList, col+"=?")
	}
	var whereCondList []string
	for _, col := range pkCols {
		whereCondList = append(whereCondList, col+"=?")
	}
	return fmt.Sprintf(updateTemplate, strings.Join(colAssignList, ", "), strings.Join(whereCondList, " and "))
}

func allColumnsExcept(except []string) []string {
	var ret []string
	for _, s := range allColumns {
		if !lib.StringInSlice(s, except) {
			ret = append(ret, s)
		}
	}
	return ret
}

func errorWithSql(err error, sqlStmt string) error {
	return errors.New(fmt.Sprintf("Error: %v\nSQL = %v", err, sqlStmt))
}

func writeRecordToDB(pk *pkey, pbody *transferRecordWOPK) error {
	exists, errRet := existsRecInDb(pk)
	if errRet != nil {
		return errRet
	}
	tx, errRet := dbconn.Begin()
	if errRet != nil {
		return errRet
	}
	defer func(perr *error) {
		if *perr != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}(&errRet)

	var sqlStmt string
	var stmt *sql.Stmt
	if exists {
		sqlStmt = makeUpdateSql(
			allColumnsExcept([]string{"started_ts_msec", "seqid", "idx", "addr"}),
			[]string{"seqid", "idx"},
		)
		stmt, errRet = dbconn.Prepare(sqlStmt)
		if errRet != nil {
			return errorWithSql(errRet, sqlStmt)
		}
		defer stmt.Close()
		_, errRet = stmt.Exec(
			pbody.seqname, pbody.lastState, pbody.last_update_msec,
			pbody.num_attaches, pbody.num_promotions,
			pbody.total_pow_duration_msec, pbody.total_tipsel_duration_msec,
			pbody.node_att, pbody.node_gtta, pbody.bundle_size, pbody.promo_bundle_size, pbody.promote_every_sec,
			pbody.force_reattach_every_min, pbody.promote_chain_yn,
			pk.seqid, pk.idx,
		)
		if errRet != nil {
			return errorWithSql(errRet, sqlStmt)
		}
	} else {
		sqlStmt = makeInsertSql(allColumns)
		stmt, errRet = dbconn.Prepare(sqlStmt)
		if errRet != nil {
			return errorWithSql(errRet, sqlStmt)
		}
		defer stmt.Close()

		_, errRet = stmt.Exec(
			pk.seqid, pk.idx, pbody.addr, pbody.seqname, pbody.lastState, pbody.started_ts_msec, pbody.last_update_msec,
			pbody.num_attaches, pbody.num_promotions,
			pbody.total_pow_duration_msec, pbody.total_tipsel_duration_msec,
			pbody.node_att, pbody.node_gtta, pbody.bundle_size, pbody.promo_bundle_size, pbody.promote_every_sec,
			pbody.force_reattach_every_min, pbody.promote_chain_yn,
		)
		if errRet != nil {
			return errorWithSql(errRet, sqlStmt)
		}
	}
	errRet = nil
	return errRet
}