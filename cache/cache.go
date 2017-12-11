// Based on https://github.com/gin-gonic/contrib/cache

package cache

import (
	"crypto/sha1"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	middleware "github.com/LYY/echo-middleware"
	"github.com/tidwall/gjson"

	"github.com/labstack/echo"
	emw "github.com/labstack/echo/middleware"
)

const (
	// DefaultExpiration default expiration second
	DefaultExpiration = 5 * time.Second
	// StayForever no expiration
	StayForever = time.Duration(-1)
	// CacheMiddlewareKey key for cache middleware
	CacheMiddlewareKey = "echo-middleware.cache"
)

var (
	PageCachePrefix = "echo-middleware.page.cache"
	ErrCacheMiss    = errors.New("cache: key not found.")
	ErrNotStored    = errors.New("cache: not stored.")
	ErrNotSupport   = errors.New("cache: not support.")

	// DefaultCacheConfig is the default cache middleware config.
	DefaultCacheConfig = Config{
		Expire:  DefaultExpiration,
		Skipper: emw.DefaultSkipper,
	}
)

// Config defines the config for cache middleware.
type Config struct {
	Expire time.Duration
	// Skipper defines a function to skip middleware.
	Skipper emw.Skipper
}

// Store defines the interface for cache store
type Store interface {
	Get(key string, value interface{}) error
	Set(key string, value interface{}, expire time.Duration) error
	Add(key string, value interface{}, expire time.Duration) error
	Replace(key string, data interface{}, expire time.Duration) error
	Delete(key string) error
	Increment(key string, data int64) (int64, error)
	Decrement(key string, data int64) (int64, error)
	Flush() error
}

type responseCache struct {
	status int
	header http.Header
	data   []byte
}

type cachedWriter struct {
	http.ResponseWriter
	status  int
	written bool
	store   Store
	expire  time.Duration
	key     string
	logger  echo.Logger
}

func urlEscape(prefix string, u string) string {
	key := url.QueryEscape(u)
	if len(key) > 200 {
		h := sha1.New()
		io.WriteString(h, u)
		key = string(h.Sum(nil))
	}
	buffer := middleware.ByteBufferPool.Get()
	defer middleware.ByteBufferPool.Put(buffer)
	buffer.WriteString(prefix)
	buffer.WriteString(":")
	buffer.WriteString(key)
	return buffer.String()
}

func jsonPostKey(prefix string, request *http.Request, keys []string) string {
	uri := request.RequestURI

	var vkeys []string
	if len(keys) > 0 {
		buf := middleware.ByteBufferPool.Get()
		defer middleware.ByteBufferPool.Put(buf)
		tee := io.TeeReader(request.Body, buf)
		bytes, _ := ioutil.ReadAll(tee)
		request.Body.Close()

		for _, k := range keys {
			vkeys = append(vkeys, gjson.GetBytes(bytes, k).String())
		}

		request.Body = ioutil.NopCloser(buf)
	}

	key := strings.Join(vkeys, ",")
	if len(key) > 200 {
		h := sha1.New()
		io.WriteString(h, uri)
		key = string(h.Sum(nil))
	}
	buffer := middleware.ByteBufferPool.Get()
	defer middleware.ByteBufferPool.Put(buffer)
	buffer.WriteString(prefix)
	buffer.WriteString(":")
	buffer.WriteString(uri)
	buffer.WriteString("?")
	buffer.WriteString(key)
	return buffer.String()
}

func newCachedWriter(store Store, expire time.Duration, responseWriter http.ResponseWriter, key string, logger echo.Logger) *cachedWriter {
	return &cachedWriter{responseWriter, 0, false, store, expire, key, logger}
}

func (w *cachedWriter) WriteHeader(code int) {
	w.status = code
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *cachedWriter) Status() int {
	return w.status
}

func (w *cachedWriter) Written() bool {
	return w.written
}

func (w *cachedWriter) Write(data []byte) (int, error) {
	ret, err := w.ResponseWriter.Write(data)
	if err == nil && w.status == http.StatusOK {
		//cache response
		store := w.store
		header := w.Header()
		// newHeader := http.Header{}
		// @TODO
		// for _, k := range header.Keys() {
		// 	newHeader.Add(k, header.Get(k))
		// }

		w.logger.Debugf("Cache Write status %d \n", w.status)
		w.logger.Debugf("Cache Write data %s \n", data)

		val := responseCache{
			200,
			header,
			data,
		}
		err = store.Set(w.key, val, w.expire)
		if err != nil {
			w.logger.Debugf("Cache Write Error %s \n", err)
			// need logger
		}
	}
	return ret, err
}

// SetCacheStore set cache store to context for next processes
func SetCacheStore(store Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(CacheMiddlewareKey, store)
			return next(c)
		}
	}
}

// JSONPostPageCache middleware for page cache with json keys
func JSONPostPageCache(config Config, keys ...string) echo.MiddlewareFunc {
	if config.Expire == 0 {
		config.Expire = DefaultCacheConfig.Expire
	}
	if config.Skipper == nil {
		config.Skipper = DefaultCacheConfig.Skipper
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			logger := c.Logger()
			logger.Debugf("Cache Begin")
			store := c.Get(CacheMiddlewareKey).(Store)
			var cache responseCache
			request := c.Request()

			key := jsonPostKey(PageCachePrefix, request, keys)
			if err := store.Get(key, &cache); err != nil {
				logger.Debugf("Cache A %s %s %s", err, key, request.RequestURI)
				// replace writer
				writer := newCachedWriter(store, config.Expire, c.Response().Writer, key, logger)
				c.Response().Writer = writer
			} else {
				logger.Debugf("Cache B")
				return c.JSONBlob(cache.status, cache.data)
				// response.WriteHeader(cache.status)
				// respHeader := response.Header()
				// for k, vals := range cache.header {
				// 	respHeader.Del(k)
				// 	common.Logger.Debugf("cache key: %s, new content: %s", k, vals)
				// 	for _, v := range vals {
				// 		respHeader.Add(k, v)
				// 	}
				// }

				// response.Write(cache.data)

				// return nil
			}
			return next(c)
		}
	}
}
