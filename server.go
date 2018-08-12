package go_redis

import (
	"net"
	"time"
	"github.com/pborman/uuid"
	"bufio"
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
	var command Command
	for {
		parts, err := ReadCommand(bur)
		if err != nil {
			logger.Errorf("bad command error %v", err)
			cli.conn.Close()
			return
		}
		if len(parts) == 0 {
			continue
		}
		stats.totalCommands += 1
		command, ok = name2command[strings.ToUpper(parts[0])]
		if ok {
			// err = command.GetFlags().Valid(parts[1:])
			// if err == nil {
			// }
			logger.Infof("in the name2command %v", parts)
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
		case "ping", "PING":
			cli.conn.Write([]byte(REPLAY_PONG))
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
