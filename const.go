package go_redis

import (
	"github.com/labstack/gommon/log"
	"errors"
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

// replay string
// first byte   + meas ok, - means error, : means int, $ means str, * means list
const (
	ERR = "-ERR"
	LINE_DELIMITER = "\r\n"
	REPLAY_OK = "+OK\r\n"
	REPLAY_ERR = "-ERR\r\n"
	REPLAY_ZERO = ":0\r\n"
	REPLAY_ONE = ":1\r\n"
	REPLAY_M1 = ":-1\r\n"
	REPLAY_M2 = ":-2\r\n"
	REPLAY_NULL_BULK = "$-1\r\n"
	REPLAY_NULL_MULTI_BULK = "*-1\r\n"
	REPLAY_EMPTY_MULTI_BULK = "*0\r\n"
	REPLAY_PONG = "+PONG\r\n"
	REPLAY_WRONG_NUMBER = "-ERR value is not an integer or out of range\r\n"
	REPLAY_WRONG_COMMAND = "-ERR no such command\r\n"
	REPLAY_WRONG_TYPE = "-ERR WRONGTYPE Operation against a key holding the wrong kind of value\r\n"
	REPLAY_WRONG_PARAMS = "-ERR wrong number of arguments for '%s' command\r\n"
)

// config const
const (
	MAXMEMORY_FLAG_LRU = 1 << 0
	MAXMEMORY_FLAG_LFU = 1 << 1
	MAXMEMORY_FLAG_ALLKEYS = 1 << 2

	MAXMEMORY_VOLATILE_LRU = 0 << 8 | MAXMEMORY_FLAG_LRU
	MAXMEMORY_VOLATILE_LFU = 1 << 8 | MAXMEMORY_FLAG_LFU
	MAXMEMORY_VOLATILE_TTL = 2 << 8
	MAXMEMORY_VOLATILE_RANDOM = 3 << 8
	MAXMEMORY_ALLKEYS_LRU  = 4 << 8 | MAXMEMORY_FLAG_LRU | MAXMEMORY_FLAG_ALLKEYS
	MAXMEMORY_ALLKEYS_LFU  = 5 << 8 | MAXMEMORY_FLAG_LFU | MAXMEMORY_FLAG_ALLKEYS
	MAXMEMORY_ALLKEYS_RANDOM = 6 << 8 | MAXMEMORY_FLAG_ALLKEYS
	MAXMEMORY_NO_EVICTION  = 7 << 8

	CONFIG_DEFAULT_MAXMEMORY_POLICY = MAXMEMORY_NO_EVICTION
)
var SYNTAX_ERROR = errors.New("syntax error")
var logger = log.New("logger")

