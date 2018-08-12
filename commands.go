package go_redis

import (
	"strconv"
	"time"
)

type CommandFlags struct {
	num    int
}

func (cf CommandFlags) GetFlags() CommandFlags {
	return cf
}

func (cf CommandFlags) Valid(parts []string) error {
	if cf.num > 0 && cf.num != len(parts) || len(parts) < cf.num {
		return SYNTAX_ERROR
	}
	return nil
}

type Command interface {
	Parse(parts []string) error
	Execute(cli *client) error
}

type ExpireCommand struct {
	Multiple int64
	key string
	ts int64
}

func (c *ExpireCommand) Parse(parts []string) error {
	if len(parts) != 2 {
		return SYNTAX_ERROR
	}
	c.key = parts[0]
	ttl, err := strconv.ParseInt(parts[1], 10, strconv.IntSize)
	if err != nil {
		return NUMBER_ERROR
	} else {
		c.ts = time.Now().UnixNano() + ttl * c.Multiple
	}
	return nil
}

func (c *ExpireCommand) Execute(cli *client) error {
	v := cli.db.Get(c.key)
	if v != nil {
		cli.db.Expire(c.key, c.ts)
		cli.rawReplay(REPLAY_ONE)
	} else {
		cli.rawReplay(REPLAY_ZERO)
	}
	return nil
}

type TTLCommand struct {
	key string
}

func (c *TTLCommand) Parse(parts []string) error {
	c.key = parts[0]
	return nil
}

func (c *TTLCommand) Execute(cli *client) error {
	v := cli.db.Get(c.key)
	if v == nil {
		cli.rawReplay(REPLAY_M2)
	} else {
		ts, ok := cli.db.GetExpire(c.key)
		if !ok {
			cli.rawReplay(REPLAY_M1)
		} else {
			cli.ReplayData(ts - time.Now().UnixNano())
		}
	}
	return nil
}

var name2command = map[string]Command{
	// TODO concurrent, we need more data, maybe we can store argv into clients!
	"GET":  &GetCommand{},
	"SET":  &SetCommand{},
	"HGET": &HGetCommand{},
	"HSET": &HSetCommand{},
	"HGETALL": &HGetAllCommand{},
	"EXPIRE": &ExpireCommand{Multiple: 1e9},
	"PEXPIRE": &ExpireCommand{Multiple: 1e6},
	"TTL": &TTLCommand{},
}
