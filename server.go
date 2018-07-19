package go_redis

import (
	"net"
	"time"
	"fmt"
	"github.com/pborman/uuid"
	"bufio"
	"strings"
	"os"
	// "encoding/binary"
	"github.com/labstack/gommon/log"
)


const (
	REDIS_RDB_VERSION = 7

	REDIS_RDB_6BITLEN = 0  // int type
	REDIS_RDB_14BITLEN = 1
	REDIS_RDB_32BITLEN = 2
	REDIS_RDB_ENCVAL = 3

	RDB_TYPE_STRING = 0
	RDB_TYPE_LIST   = 1
	RDB_TYPE_SET    = 2
	RDB_TYPE_ZSET   = 3
	RDB_TYPE_HASH   = 4
	RDB_TYPE_ZSET_2 = 5 /* ZSET version 2 with doubles stored in binary. */
	RDB_TYPE_MODULE = 6
	RDB_TYPE_MODULE_2 = 7


	REDIS_RDB_OPCODE_EXPIRETIME_MS = 252
	REDIS_RDB_OPCODE_EXPIRETIME = 253
	REDIS_RDB_OPCODE_SELECTDB = 254
	REDIS_RDB_OPCODE_EOF = 255

)

var logger = log.New("logger")

type RedisServer struct {

}

type Set map[string]bool

type redisDb struct {
	tmpDict map[string]string // just for first implementation
	dict Set
	expires map[string]uint64
	blocking_keys Set
	ready_keys Set
	watched_keys Set
	// evitction_pool evictionPoolEntry
	id uint8
	avg_ttl uint64
}

type redisCommand interface {
	redisCommandProc(c *client) interface{}
}

// listen 进程为每个 客户端fork 出来的 client;
type client struct {
	// 链接描述
	id string
	file_descriptor int
	conn	net.Conn
	db *redisDb
	// 当前命令描述
	argv []string
	cmd, lastcmd *redisCommand
	// 返回值
	reply []interface{}
	reply_bytes uint64
}


func (cli *client) serve() {
	bur := bufio.NewReader(cli.conn)
	defer func () {
		cli.conn.Close()
	}()
	for {
		cmd, err := bur.ReadString(byte('\n'))
		if err != nil {
			logger.Errorf("buffer read error %v", err)
			break
		}
		// cmd parser, and validate params
		parts := strings.Split(strings.TrimSpace(cmd), " ")
		logger.Debug("%s: the cmd get is: %v\n", cli.id, parts)
		switch parts[0] {
		case "get":
			v, ok := cli.db.tmpDict[parts[1]]
			if !ok {
				logger.Errorf("string key %s not exist\n", parts[1])
			}
			cli.conn.Write([]byte(v))
			cli.conn.Write([]byte("\n\r"))
		case "set":
			cli.db.tmpDict[parts[1]] = parts[2]
			cli.conn.Write([]byte("ok\n\r"))
		case "exit":
			logger.Debug("exit cmd from client\n")
			cli.conn.Write([]byte("closed\n\r"))
			break
		case "keys":
			logger.Debugf("show keys %v\n", cli.db.tmpDict)
			cli.conn.Write([]byte("show keys\n\r"))
		}
	}
}

type redisServer struct {
	pid uint64
	configfile string
	executable string
	exec_argv []string
	db *redisDb  // 第一版只实现一个db
	requirepass string
	pidfile string
	// tcp 配置
	port int
	bindaddrs []string
	unixsocket string
	unixsocketperm string
	ipfds []int
	sofd int
	clients []client
	current_client *client
	client_paused bool

	next_client_id uint64
}

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

func (srv *redisServer) dump() {
	fo, err := os.Create("go_redis.rdb")
	r := rdbWriter{fo}
	defer fo.Close()
	if err != nil {
		logger.Error(err)
		return
	}
	r.WriteString(fmt.Sprintf("REDIS%04d", REDIS_RDB_VERSION))
	// binary.Write(fo, binary.LittleEndian, uint8(REDIS_RDB_OPCODE_SELECTDB))
	r.WriteFlag(REDIS_RDB_OPCODE_SELECTDB)
	r.WriteLen(0)
	// rdbSaveKeyValuePair
	for k, v := range srv.db.tmpDict {
		// binary.Write(fo, binary.LittleEndian, uint8(REDIS_RDB_OPCODE_SELECTDB))
		r.WriteFlag(RDB_TYPE_STRING)
		r.WRDBString(k)
		r.WRDBString(v)
	}
	// EOF
	r.WriteFlag(REDIS_RDB_OPCODE_EOF)
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

func (r *rdbReader) ReadFlag() byte {
	return r.Read(1)[0]
}

func (srv *redisServer) loadRdb() {
	fo, err := os.Open("go_redis.rdb")
	if err != nil {
		logger.Error(err)
		return
	}
	defer fo.Close()
	var flag byte
	var l int
	var key, value string

	r := rdbReader{f: fo, rbuf: make([]byte, 128)}
	tmp := r.Read(9)
	logger.Debug(string(tmp))
	flag = r.ReadFlag()
	logger.Debugf("flag %d", flag)
	logger.Debug(flag)
	r.ReadLen()

	cnt := 0
	for {
		flag = r.ReadFlag()
		logger.Debugf("flag %d", flag)
		if flag == REDIS_RDB_OPCODE_EOF {
			break
		}
		l = r.ReadLen()
		key = r.ReadString(l)
		logger.Debugf("len: %d key %s", l, key)
		switch flag {
		case RDB_TYPE_STRING:
			l = r.ReadLen()
			value = r.ReadString(l)
			logger.Debugf("string value %d, %s", l, value)
			srv.db.tmpDict[key] = value
		}
		cnt += 1
		if cnt > 10 {
			break
		}
	}
}

func (srv *redisServer) newConn(conn net.Conn) (cli client) {
	cli = client{
		id: uuid.New(),
		db: srv.db,
		conn: conn,
	}
	return
}

func (srv *redisServer) run() error {
	// listen from 8989
	hostPort := ":8989"
	l, err := net.Listen("tcp", hostPort)
	if err != nil {
		logger.Errorf("listen socket error %s %v", hostPort, err)
		return err
	}
	go func () {
		for {
			time.Sleep(10 * time.Second)
			logger.Debug("save dump")
			srv.dump()
		}
	}()
	tempDelay := 100 * time.Millisecond
	for {
		rw, e := l.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				logger.Errorf("http: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		c := srv.newConn(rw)
		go c.serve()

	}
	return nil
}


func Start() {
	logger.SetLevel(log.DEBUG)
	var t redisDb = redisDb{tmpDict: map[string]string{}}
	srv := redisServer{
		db: &t,
	}
	logger.Debug("load db start")
	srv.loadRdb()
	logger.Debug("load db ok")
	err := srv.run()
	if err != nil {
		logger.Errorf("start err %v", err)
	}
}
