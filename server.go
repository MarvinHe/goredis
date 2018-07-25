package go_redis

import (
	"net"
	"time"
	"github.com/pborman/uuid"
	"bufio"
	"strconv"
	"github.com/labstack/gommon/log"
	"strings"
)

type RedisServer struct {
}
var stats redisStats

type Set map[string]bool

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
	var hash Hash
	var command Command
	for {
		now = time.Now().UnixNano()
		parts, err := ReadCommand(bur)
		if err != nil {
			logger.Errorf("bad command error %v", err)
			cli.conn.Close()
			return
		}
		stats.totalCommands += 1
		command, ok = name2command[strings.ToUpper(parts[0])]
		if ok {
			err = command.Parse(parts[1:])
			if err != nil {
				cli.conn.Write([]byte(ErrorReply(err.Error())))
			} else {
				command.Execute(cli)
			}
			continue
		}
		logger.Debugf("%s: the cmd get is: %v", cli.id, parts)
		switch parts[0] {
		case "EXPIRE":
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
		case "PEXPIRE":
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
		case "TTL":
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
		case "ping", "PING":
			cli.conn.Write([]byte(REPLAY_PONG))
		case "hset", "HSET":
			hash, ok = cli.db.hashDict[parts[1]]
			if !ok {
				hash = map[string]string{}
				cli.db.hashDict[parts[1]] = hash
			}
			hash[parts[2]] = parts[3]
			cli.conn.Write([]byte(REPLAY_ONE))
		case "hget", "HGET":
			hash, ok = cli.db.hashDict[parts[1]]
			if ok {
				v, ok = hash[parts[2]]
			}
			if ok {
				cli.conn.Write([]byte(EncodeReply(v)))
			} else {
				cli.conn.Write([]byte(REPLAY_NULL_BULK))
			}
		case "hgetall", "HGETALL":
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
	client_paused bool

	next_client_id uint64
	config      Config
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
			srv.dump()
		}
	}()
	go func () {
		for {
			// for debug, sleep longer
			time.Sleep(100 * time.Second)
			srv.activeExpire()
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
		stats.totalConns += 1
		logger.Infof("new connection: %s created", c.id)
		go c.serve()

	}
	return nil
}

func Start() {
	logger.SetLevel(log.DEBUG)
	var t redisDb = redisDb{
		tmpDict: map[string]string{},
		hashDict: map[string]Hash{},
		dict: map[string]robji{},
		expires: map[string]int64{},
	}
	srv := redisServer{
		db: &t,
		config: Config{},
	}
	stats = redisStats{}
	logger.Debug("load db start")
	srv.loadRdb()
	logger.Debug("load db ok")
	err := srv.run()
	if err != nil {
		logger.Errorf("start err %v", err)
	}
}
