package go_redis

import (
	"net"
	"time"
	"github.com/pborman/uuid"
	"bufio"
	// "strings"
	"strconv"
	"github.com/labstack/gommon/log"
)

type RedisServer struct {

}

type Set map[string]bool

type redisDb struct {
	tmpDict map[string]string // just for first implementation
	hashDict map[string]map[string]string
	dict Set
	expires map[string]int64 // expires in millisecond
	blocking_keys Set
	ready_keys Set
	watched_keys Set
	// evitction_pool evictionPoolEntry
	id uint8
	avg_ttl int64
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
	var ok bool
	var v string
	var ts, now int64
	var hash map[string]string
	for {
		now = time.Now().UnixNano()
		parts, err := ReadCommand(bur)
		if err != nil {
			logger.Errorf("bad command error %v", err)
			cli.conn.Close()
			return
		}
		logger.Debugf("%s: the cmd get is: %v", cli.id, parts)
		switch parts[0] {
		case "get":
			v, ok = cli.db.tmpDict[parts[1]]
			if !ok {
				cli.conn.Write([]byte(REPLAY_NULL_BULK))
				continue
			}
			ts, ok = cli.db.expires[parts[1]]
			if ok && ts < now {  // expired key
				delete(cli.db.tmpDict, parts[1])
				delete(cli.db.expires, parts[1])
				cli.conn.Write([]byte(REPLAY_NULL_BULK))
				continue
			}
			cli.conn.Write([]byte(EncodeReply(v)))
		case "set":
			cli.db.tmpDict[parts[1]] = parts[2]
			cli.conn.Write([]byte(REPLAY_OK))
		case "exit":
			logger.Debug("exit cmd from client\n")
			cli.conn.Write([]byte("closed\n\r"))
			break
		case "expire":
			if _, ok = cli.db.tmpDict[parts[1]]; !ok {  // no such key
				cli.conn.Write([]byte(REPLAY_ZERO))
				continue
			}
			// store ttl in nanoseconds
			ttl, err := strconv.ParseInt(parts[2], 10, strconv.IntSize)
			if err != nil {
				logger.Errorf("ttl error: %s, %s %v", parts[1], parts[2], err)
				cli.conn.Write([]byte(REPLAY_WRONG_NUMBER))
				continue
			}
			cli.db.expires[parts[1]] = now + ttl * 1e9
			cli.conn.Write([]byte(REPLAY_ONE))
		case "pexpire":
			if _, ok = cli.db.tmpDict[parts[1]]; !ok {  // no such key
				cli.conn.Write([]byte(REPLAY_ZERO))
				continue
			}
			ttl, err := strconv.ParseInt(parts[2], 10, strconv.IntSize)
			if err != nil {
				logger.Errorf("ttl error: %s, %s %v", parts[1], parts[2], err)
				cli.conn.Write([]byte(REPLAY_WRONG_NUMBER))
				continue
			}
			cli.db.expires[parts[1]] = now + ttl * 1e6
			cli.conn.Write([]byte(REPLAY_ONE))
		case "ttl":
			if _, ok = cli.db.tmpDict[parts[1]]; !ok {  // no such key
				cli.conn.Write([]byte(REPLAY_M2))
			} else {
				ts, ok = cli.db.expires[parts[1]]
				ts -= now
				if ok && ts > 0 {
					cli.conn.Write([]byte(EncodeReply(ts / 1e9)))
				} else {
					logger.Errorf("expire key %s not exist\n", parts[1])
					cli.conn.Write([]byte(REPLAY_M1))
				}
			}
		case "keys":
			logger.Debugf("show keys %v\n", cli.db.tmpDict)
			cli.conn.Write([]byte("show keys\n\r"))
		case "ping", "PING":
			cli.conn.Write([]byte(REPLAY_PONG))
		case "hset":
			hash, ok = cli.db.hashDict[parts[1]]
			if !ok {
				hash = map[string]string{}
				cli.db.hashDict[parts[1]] = hash
			}
			hash[parts[2]] = parts[3]
			cli.conn.Write([]byte(REPLAY_ONE))
		case "hget":
			hash, ok = cli.db.hashDict[parts[1]]
			if ok {
				v, ok = hash[parts[2]]
			}
			if ok {
				cli.conn.Write([]byte(EncodeReply(v)))
			} else {
				cli.conn.Write([]byte(REPLAY_NULL_BULK))
			}
		case "hgetall":
			hash, ok = cli.db.hashDict[parts[1]]
			if !ok {
				cli.conn.Write([]byte(REPLAY_EMPTY_MULTI_BULK))
				continue
			}
			ts, ok = cli.db.expires[parts[1]]
			if ok && ts < now {  // expired key
				delete(cli.db.hashDict, parts[1])
				delete(cli.db.expires, parts[1])
				cli.conn.Write([]byte(REPLAY_EMPTY_MULTI_BULK))
				continue
			}
			cli.conn.Write([]byte(EncodeReply(hash)))
		default:
			cli.conn.Write([]byte(REPLAY_WRONG_COMMAND))
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
		logger.Infof("new connection: %s created", c.id)
		go c.serve()

	}
	return nil
}

func Start() {
	logger.SetLevel(log.DEBUG)
	var t redisDb = redisDb{
		tmpDict: map[string]string{},
		expires: map[string]int64{},
		hashDict: map[string]map[string]string{},
	}
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
