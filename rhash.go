package go_redis


type Hash map[string]string
type RHash struct {
	robjBase
	hash Hash
}

func (h *RHash) GetField(key string) string {
	return h.hash[key]
}

func (h *RHash) SetField(key, value string) error {
	h.hash[key] = value
	return nil
}

func (h *RHash) GetAll() Hash {
	return h.hash
}

func CreateRHash(h Hash) *RHash {
	return &RHash{robjBase{RDB_TYPE_HASH << 4, 0, 0}, h}
}
/**
type Command interface {
	Parse(parts []string) error
	Execute(cli *client) error
}
*/

type HGetCommand struct {
	key   string
	field string
}

func (c *HGetCommand) Parse(parts []string) error {
	if len(parts) != 2 {
		return SYNTAX_ERROR
	}
	c.key, c.field = parts[0], parts[1]
	return nil
}

func (c *HGetCommand) Execute(cli *client) error {
	h := cli.db.Get(c.key)
	if h != nil && h.GetType() != RDB_TYPE_HASH {
		cli.conn.Write([]byte(REPLAY_WRONG_TYPE))
		return nil
	}
	if h == nil {
		cli.conn.Write([]byte(REPLAY_NULL_BULK))
	} else {
		v := h.(*RHash).GetField(c.field)
		if v == "" {
			cli.conn.Write([]byte(REPLAY_NULL_BULK))
		} else {
			cli.conn.Write([]byte(EncodeReply(v)))
		}
	}
	return nil
}


type HSetCommand struct {
	key   string
	field string
	value string
}

func (c *HSetCommand) Parse(parts []string) error {
	if len(parts) != 3 {
		return SYNTAX_ERROR
	}
	c.key, c.field, c.value = parts[0], parts[1], parts[2]
	return nil
}

func (c *HSetCommand) Execute(cli *client) error {
	h := cli.db.Get(c.key)
	if h != nil && h.GetType() != RDB_TYPE_HASH {
		cli.rawReplay(REPLAY_WRONG_TYPE)
		return nil
	}
	if h != nil {
		h.(*RHash).SetField(c.field, c.value)
	} else {
		cli.db.Set(c.key, CreateRHash(Hash{c.field: c.value}))
	}
	cli.conn.Write([]byte(REPLAY_OK))
	return nil
}

type HGetAllCommand struct {
	key    string
}

func (c *HGetAllCommand) Parse(parts []string) error {
	if len(parts) != 1 {
		return SYNTAX_ERROR
	}
	return nil
}


func (c *HGetAllCommand) Execute(cli *client) error {
	h := cli.db.Get(c.key)
	if h == nil {
		cli.rawReplay(REPLAY_EMPTY_MULTI_BULK)
	} else if h.GetType() != RDB_TYPE_HASH {
		cli.rawReplay(REPLAY_WRONG_TYPE)
	} else {
		cli.ReplayData(h.(*RHash).GetAll())
	}
	return nil
}
