package go_redis

import "github.com/labstack/gommon/log"

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
)

var logger = log.New("logger")

