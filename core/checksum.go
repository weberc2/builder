package core

import (
	"encoding/binary"
	"hash/adler32"
)

func ChecksumBytes(bs []byte) uint32 { return adler32.Checksum(bs) }

func ChecksumString(s string) uint32 { return ChecksumBytes([]byte(s)) }

func JoinChecksums(checksums ...uint32) uint32 {
	buf := make([]byte, len(checksums)*4)
	for i, checksum := range checksums {
		binary.BigEndian.PutUint32(buf[i*4:i*4+4], checksum)
	}
	return ChecksumBytes(buf)
}
