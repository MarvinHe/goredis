package go_redis

import (
	"os"
	"encoding/binary"
)

type rdbWriter struct {
	f *os.File
}

func encodeLen(i int) []byte {
	buf := []byte{}
	if i < 1 << 6 {
		buf = append(buf, byte(i&0xFF | REDIS_RDB_6BITLEN << 6))
	}
	return buf
}

func (r *rdbWriter) WriteFlag(flag uint8) {
	r.f.Write([]byte{byte(flag)})
}

func (r *rdbWriter) WriteLen(i int) {
	r.f.Write(encodeLen(i))
}

func (r *rdbWriter) WriteString(s string) {
	r.f.WriteString(s)
}

func (r *rdbWriter) WRDBString(s string) {
	r.f.Write(encodeLen(len(s)))
	r.f.WriteString(s)
}

func (r *rdbWriter) WriteInt(i interface{}) {
	binary.Write(r.f, binary.LittleEndian, i)
}

type rdbReader struct {
	f *os.File
	rbuf []byte
}

func (r *rdbReader) ReadLen() int {
	r.f.Read(r.rbuf[:1])
	switch r.rbuf[0] >> 6 {
	case 0:
		return int(r.rbuf[0] & 0x3f)
	case 1:
		r.f.Read(r.rbuf[1:2])
		return int((r.rbuf[0] & 0x3f) << 8) +  int(r.rbuf[1])
	}
	return 0
}

func (r *rdbReader) Read(n int) []byte {
	r.f.Read(r.rbuf[:n])
	return r.rbuf[:n]
}

func (r *rdbReader) ReadString(n int) string {
	r.f.Read(r.rbuf[:n])
	return string(r.rbuf[:n])
}

func (r *rdbReader) ReadFlag() byte {
	return r.Read(1)[0]
}

func (r *rdbReader) ReadInt64() (i int64) {
	binary.Read(r.f, binary.LittleEndian, &i)
	return
}
