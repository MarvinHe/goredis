package go_redis_test

import (
	"testing"
	"go_redis"
	"fmt"
)

func TestParser(t *testing.T) {
	var line []byte
	line = []byte("get abc")
	parts := go_redis.ParseLine(line)
	t.Log(line)
	t.Log(parts)
	if parts[0] != "get" || parts[1] != "abc" {
		t.Errorf("parse error %s %v", string(line), parts)
	}
}

func TestEncodeReplay(t *testing.T) {
	var line string = "ooa"
	reply := go_redis.EncodeReply(line)
	if reply != "$3\r\nooa\r\n" {
		t.Errorf("encode reply error %s %s", line, reply)
	}
}

func TestMapOFMap(t *testing.T) {
	hashDict := map[string]map[string]string{}
	hashDict["a"] = map[string]string{"key": "value"}
	fmt.Println(hashDict)
	t.Error(hashDict)
}
