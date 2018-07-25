package go_redis

import (
	"os"
	"encoding/binary"
	"fmt"
	"time"
)

type rdbWriter struct {
	f *os.File
}

func encodeLen(i int) []byte {
	buf := []byte{}
	if i < 1 << 6 {
		buf = append(buf, byte(i&0xFF | REDIS_RDB_6BITLEN << 6))
	}
	return buf
}

func (r *rdbWriter) WriteFlag(flag uint8) {
	r.f.Write([]byte{byte(flag)})
}

func (r *rdbWriter) WriteLen(i int) {
	r.f.Write(encodeLen(i))
}

func (r *rdbWriter) WriteString(s string) {
	r.f.WriteString(s)
}

func (r *rdbWriter) WRDBString(s string) {
	r.f.Write(encodeLen(len(s)))
	r.f.WriteString(s)
}

func (r *rdbWriter) WriteInt(i interface{}) {
	binary.Write(r.f, binary.LittleEndian, i)
}

type rdbReader struct {
	f *os.File
	rbuf []byte
}

func (r *rdbReader) ReadLen() int {
	r.f.Read(r.rbuf[:1])
	switch r.rbuf[0] >> 6 {
	case 0:
		return int(r.rbuf[0] & 0x3f)
	case 1:
		r.f.Read(r.rbuf[1:2])
		return int((r.rbuf[0] & 0x3f) << 8) +  int(r.rbuf[1])
	}
	return 0
}

func (r *rdbReader) Read(n int) []byte {
	r.f.Read(r.rbuf[:n])
	return r.rbuf[:n]
}

func (r *rdbReader) ReadString(n int) string {
	r.f.Read(r.rbuf[:n])
	return string(r.rbuf[:n])
}

func (r *rdbReader) ReadEncodeString() string {
	return r.ReadString(r.ReadLen())
}

func (r *rdbReader) ReadFlag() byte {
	return r.Read(1)[0]
}

func (r *rdbReader) ReadInt64() (i int64) {
	binary.Read(r.f, binary.LittleEndian, &i)
	return
}

func (srv *redisServer) dump() {
	logger.Debug("save dump")
	// TODO we need lock when dumping
	now := time.Now().UnixNano()
	fo, err := os.Create("go_redis.rdb")
	r := rdbWriter{fo}
	defer fo.Close()
	if err != nil {
		logger.Error(err)
		return
	}
	r.WriteString(fmt.Sprintf("REDIS%04d", REDIS_RDB_VERSION))
	r.WriteFlag(REDIS_RDB_OPCODE_SELECTDB)
	r.WriteLen(0)
	// rdbSaveKeyValuePair
	for k, v := range srv.db.tmpDict {
		expire, ok := srv.db.expires[k]
		if ok {
			if expire < now {   // expired key, do not delete, leave it to background cron
				// delete(srv.db.expires, k)
				// delete(srv.db.tmpDict, k)
				continue
			}
			r.WriteFlag(REDIS_RDB_OPCODE_EXPIRETIME_MS)
			r.WriteInt(expire / 1e6)
		}
		r.WriteFlag(RDB_TYPE_STRING)
		r.WRDBString(k)
		r.WRDBString(v)
	}
	for k, v := range srv.db.hashDict {
		expire, ok := srv.db.expires[k]
		if ok {
			if expire < now {   // expired key, do not delete, leave it to background cron
				// delete(srv.db.expires, k)
				// delete(srv.db.hashDict, k)
				continue
			}
			r.WriteFlag(REDIS_RDB_OPCODE_EXPIRETIME_MS)
			r.WriteInt(expire / 1e6)
		}
		r.WriteFlag(RDB_TYPE_HASH)
		r.WRDBString(k)
		r.WriteLen(len(v))
		for key, value := range v {
			r.WRDBString(key)
			r.WRDBString(value)
		}
	}
	// EOF
	r.WriteFlag(REDIS_RDB_OPCODE_EOF)
}

func (srv *redisServer) loadRdb() {
	fo, err := os.Open("go_redis.rdb")
	if err != nil {
		logger.Error(err)
		return
	}
	defer fo.Close()
	var flag byte
	var key, value string
	var ts int64
	var l int
	var hash map[string]string
	var valid bool
	now := time.Now().UnixNano()

	r := rdbReader{f: fo, rbuf: make([]byte, 128)}
	r.Read(9)
	r.ReadFlag()
	r.ReadLen()

	cnt := 0
	for {
		valid = true
		ts = 0
		flag = r.ReadFlag()
		logger.Debugf("flag %d", flag)
		if flag == REDIS_RDB_OPCODE_EOF {
			break
		}
		if flag == REDIS_RDB_OPCODE_EXPIRETIME_MS {
			ts = r.ReadInt64() * 1e6
			flag = r.ReadFlag()
			if ts < now {
				valid = false
			}
		}
		key = r.ReadEncodeString()
		fmt.Printf("ts %s is: %d %s", key, ts, valid)
		switch flag {
		case RDB_TYPE_STRING:
			value = r.ReadEncodeString()
			logger.Debugf("string value %d, %s", len(value), value)
			if valid {
				srv.db.tmpDict[key] = value
			}
		case RDB_TYPE_HASH:
			l = r.ReadLen()
			hash = map[string]string{}
			for i := 0; i < l; i += 1 {
				hash[r.ReadEncodeString()] = r.ReadEncodeString()
			}
			if valid {
				srv.db.hashDict[key] = hash
			}
		}
		if ts > now {
			srv.db.expires[key] = ts
		}
		// just for debug
		cnt += 1
		if cnt > 200 {
			break
		}
	}
}

