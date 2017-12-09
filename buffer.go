package middleware

import (
	"bytes"
	"sync"
)

type byteBufferPool struct {
	pool *sync.Pool
}

var (
	// ByteBufferPool byte buffer pool
	ByteBufferPool = &byteBufferPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
)

// Get get bytes.Buffer
func (p *byteBufferPool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

func (p *byteBufferPool) Put(buffer *bytes.Buffer) {
	if buffer != nil {
		buffer.Reset()
		p.pool.Put(buffer)
	}
}
