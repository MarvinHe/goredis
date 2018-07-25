package go_redis

type redisStats struct {
	totalConns       int64
	totalCommands    int64
	OPS	         int64
	totalNetInBytes  int64
	totalNetOutBytes int64
	rejectedConns    int64
	expiredKeys      int64
	evictedKeys      int64

	totalMiss        int64
	totalHits        int64
}
