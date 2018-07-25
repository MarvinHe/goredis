package go_redis

import "time"

type redisDb struct {
	tmpDict map[string]string // just for first implementation
	hashDict map[string]Hash
	dict map[string]robji
	expires map[string]int64 // expires in millisecond
	blocking_keys Set
	ready_keys Set
	watched_keys Set
	// evitction_pool evictionPoolEntry
	id uint8
	avg_ttl int64
}

func (db *redisDb) expireIfNeeded(key string) (expired bool) {
	now := time.Now().UnixNano()
	if ts, ok := db.expires[key]; ok && ts < now {
		delete(db.expires, key)
		delete(db.tmpDict, key)
		delete(db.hashDict, key)
		stats.expiredKeys += 1
		expired = true
	}
	return
}

func (db *redisDb) ValidKey(key string) bool {
	db.expireIfNeeded(key)
	_, ok := db.dict[key]
	return ok
}

func (db *redisDb) get(key string) robji {
	o, ok := db.dict[key]
	if !ok {
		return nil
	}
	o.UpdateLRU()
	return o
}

func (db *redisDb) Get(key string) (o robji) {
	if db.expireIfNeeded(key) {
		return
	}
	o = db.get(key)
	if o == nil {
		stats.totalMiss += 1
	} else {
		stats.totalHits += 1
	}
	return o
}

func (db *redisDb) GetForWrite(key string) robji {
	// TODO: ReadWrite Lock
	db.expireIfNeeded(key)
	return db.get(key)
}

func (db *redisDb) Set(key string, obj robji) error {
	// TODO: concurrent & lock & consider type, if key exists in dict
	db.dict[key] = obj
	return nil
}

func (db *redisDb) Expire(key string, ts int64) error {
	db.expires[key] = ts
	return nil
}

func (db *redisDb) SetExpire(key string, obj robji, ts int64) error {
	// TODO: concurrent & lock & consider type, if key exists in dict
	db.dict[key] = obj
	db.expires[key] = ts
	return nil
}
