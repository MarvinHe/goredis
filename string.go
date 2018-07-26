package go_redis

import (
	"fmt"
	"strconv"
)

type RString struct {
	robjBase
	s string
}

func (r *RString) Get() string {
	return r.s
}

func (r *RString) Set(s string) {
	// TODO: change lru flag
	r.s = s
	r.UpdateLRU()
	return
}

func CreateRString(s string) *RString {
	return &RString{robjBase{RDB_TYPE_STRING << 4, 0, 0}, s}
}

// Commands
type GetCommand struct {
	key string
}

func (c *GetCommand) Parse(parts []string) error {
	fmt.Println("get command parts is: ", parts, len(parts), len(parts[0]))
	if len(parts) != 1 {
		return SYNTAX_ERROR
	} else {
		c.key = parts[0]
	}
	return nil
}

func (c *GetCommand) Execute(cli *client) error {
	v := cli.db.Get(c.key)
	if v == nil {
		cli.conn.Write([]byte(REPLAY_NULL_BULK))
	} else if v.GetType() != RDB_TYPE_STRING {
		cli.conn.Write([]byte(REPLAY_WRONG_TYPE))
	} else {
		cli.conn.Write([]byte(EncodeReply(v.(*RString).Get())))
	}
	return nil
}

type SetCommand struct {
	key, value string
	expire int64
	nx, xx bool
}

func (c *SetCommand) Parse(parts []string) error {
	c.nx = false
	c.xx = false
	c.expire = 0
	l := len(parts)
	if l < 1 || l > 5 {
		return SYNTAX_ERROR
	}
	c.key = parts[0]
	c.value = parts[1]
	if l > 3 {
		fmt.Println("set command parts px, ex is: ", parts, len(parts), len(parts[0]), parts[1])
		var multiply int64
		if parts[2] == "EX" {
			multiply = 1e9
		} else if parts[2] == "PX" {
			multiply = 1e6
		} else {
			return SYNTAX_ERROR
		}
		ttl, err := strconv.ParseInt(parts[3], 10, strconv.IntSize)
		if err != nil {
			return err
		}
		// c.expire = time.Now().Nanosecond() + ttl * multiply
		c.expire = ttl * multiply
	}
	if l % 2 == 1 {
		if parts[l-1] == "NX" {
			c.nx = true
		} else if parts[l-1] == "XX" {
			c.xx = true
		} else {
			return SYNTAX_ERROR
		}
	}
	return nil
}

func (c *SetCommand) Execute(cli *client) error {
	v := cli.db.Get(c.key)
	if v != nil && v.GetType() != RDB_TYPE_STRING {
		cli.conn.Write([]byte(REPLAY_WRONG_TYPE))
		return nil
	}
	if v != nil && c.xx || v == nil && c.nx || !c.xx && !c.nx {
		fmt.Println("in setting ", c.key, c.value, c.expire)
		if v != nil {
			v.(*RString).Set(c.value)
		} else {
			if c.expire > 0 {
				cli.db.SetExpire(c.key, CreateRString(c.value), c.expire)
			} else {
				cli.db.Set(c.key, CreateRString(c.value))
			}
		}
		cli.conn.Write([]byte(REPLAY_OK))
	} else {
		cli.conn.Write([]byte(REPLAY_M1))
	}
	return nil
}
