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

/**
 * scan & delete expire keys
 */
func (srv *redisServer) activeExpire() {
	fmt.Println("active expire")
	var toScan bool = true
	var i, delCnt int
	var key string
	var now int64 = time.Now().UnixNano()
	endTime := now + SCAN_TIMELIMIT
	for toScan {
		delCnt = 0
		for i = 0; i < SCAN_BATCH; i += 1 {
			key = randomKey(srv.db)
			if srv.db.expireIfNeeded(key) {
				delCnt += 1
			}
		}
		toScan = delCnt > SCAN_BATCH / 4
		now = time.Now().UnixNano()
		if now > endTime { // exceed scan timelimit
			break
		}
	}
}
