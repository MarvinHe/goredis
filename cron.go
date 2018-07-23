package go_redis

import (
	"time"
	"fmt"
)

const SCAN_BATCH = 100
const SCAN_TIMELIMIT = 1e6  // once scan at most 1 millisecond

func randomKey(db *redisDb) string {
	return ""
}

func (db *redisDb) expireKey(key string) bool {
	delete(db.expires, key)
	delete(db.tmpDict, key)
	delete(db.hashDict, key)
	return true
}
/**
 * scan & delete expire keys
 */
func (srv *redisServer) activeExpire() {
	fmt.Println("active expire")
	var toScan bool = true
	var i, delCnt int
	var key string
	var now int64 = time.Now().Nanosecond()
	endTime := now + SCAN_TIMELIMIT
	for toScan {
		delCnt = 0
		for i = 0; i < SCAN_BATCH; i += 1 {
			key = randomKey(srv.db)
			if srv.db.expires[key] < now {
				srv.db.expireKey(key)
				delCnt += 1
			}
		}
		toScan = delCnt > SCAN_BATCH / 4
		now = time.Now().Nanosecond()
		if now > endTime { // exceed scan timelimit
			break
		}
	}
}
