package go_redis


type Command interface {
	Parse(parts []string) error
	Execute(cli *client) error
}


var name2command = map[string]Command{
	"GET":  &GetCommand{},
	"SET":  &SetCommand{},
	"HGET": &HGetCommand{},
	"HSET": &HSetCommand{},
}
