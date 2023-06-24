/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
*/
package istoragecache

import (
	"bytes"
	"fmt"
	"github.com/VictoriaMetrics/fastcache"
	go_object_pool "github.com/theodesp/go-object-pool"
	"github.com/valyala/bytebufferpool"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"testing"
)

/*
Before:
goos: linux
goarch: amd64
pkg: github.com/voedger/voedger/pkg/istoragecache
cpu: 12th Gen Intel(R) Core(TM) i7-12700
BenchmarkAppStorage_Metrics
BenchmarkAppStorage_Metrics/GET
BenchmarkAppStorage_Metrics/GET-20         	 4889203	       239.9 ns/op	       8 B/op	       1 allocs/op
BenchmarkAppStorage_Metrics/GET-20         	11087750	       102.1 ns/op	       8 B/op	       1 allocs/op

After:
TBD
*/
func BenchmarkAppStorage_Metrics(b *testing.B) {
	testData := []byte("atestdata")
	ts := &testStorage{get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
		*data = testData
		return true, nil
	}}
	tsp := &testStorageProvider{storage: ts}
	cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "vvm")
	storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
	if err != nil {
		panic(err)
	}
	pk := []byte("pk")
	cc := []byte("cc")
	var res []byte

	b.Run("GET", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ok, err := storage.Get(pk, cc, &res)
			if !ok {
				panic("not ok")
			}
			if err != nil {
				panic(err)
			}

		}
	})
}

func Benchmark_Get_FastCache(b *testing.B) {
	c := fastcache.New(testCacheSize)
	k := []byte("pkcc")
	v := []byte("atestdata")

	c.Set(k, v)

	var buf []byte

	for i := 0; i < b.N; i++ {
		buf = c.Get(buf[:0], k)
		if string(buf) != string(v) {
			panic(fmt.Errorf("BUG: invalid value obtained; got %q; want %q", buf, v))
		}
	}
}
func Benchmark_Get_IStorageCache(b *testing.B) {
	pk := []byte("pk")
	cc := []byte("cc")
	v := []byte("atestdata")
	var res []byte

	ts := &testStorage{get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
		*data = v
		return true, nil
	}}
	tsp := &testStorageProvider{storage: ts}
	cachingStorageProvider := Provide(testCacheSize, tsp, imetrics.Provide(), "vvm")
	storage, err := cachingStorageProvider.AppStorage(istructs.AppQName_test1_app1)
	if err != nil {
		panic(err)
	}

	for i := 0; i < b.N; i++ {
		ok, err := storage.Get(pk, cc, &res)
		if err != nil {
			panic(err)
		}
		if !ok {
			panic("not ok")
		}
		if string(res) != string(v) {
			panic(fmt.Errorf("BUG: invalid value obtained; got %q; want %q", res, v))
		}
	}
}
func Benchmark_Get_IStorageCache_WithoutMetrics(b *testing.B) {
	pk := []byte("pk")
	cc := []byte("cc")
	v := []byte("atestdata")
	var res []byte

	ts := &testStorage{get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
		*data = v
		return true, nil
	}}

	storage := &storageCacheWithoutMetrics{
		cache:   fastcache.New(testCacheSize),
		storage: ts,
	}

	for i := 0; i < b.N; i++ {
		ok, err := storage.Get(pk, cc, &res)
		if err != nil {
			panic(err)
		}
		if !ok {
			panic("not ok")
		}
		if string(res) != string(v) {
			panic(fmt.Errorf("BUG: invalid value obtained; got %q; want %q", res, v))
		}
	}
}
func Benchmark_Get_IStorageCache_WithoutMetrics_WithByteBufferPool(b *testing.B) {
	pk := []byte("pk")
	cc := []byte("cc")
	v := []byte("atestdata")
	var res []byte

	ts := &testStorage{get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
		*data = v
		return true, nil
	}}

	storage := &storageCacheWithoutMetricsWithByteBufferPool{
		cache:   fastcache.New(testCacheSize),
		storage: ts,
	}

	for i := 0; i < b.N; i++ {
		ok, err := storage.Get(pk, cc, &res)
		if err != nil {
			panic(err)
		}
		if !ok {
			panic("not ok")
		}
		if string(res) != string(v) {
			panic(fmt.Errorf("BUG: invalid value obtained; got %q; want %q", res, v))
		}
	}
}
func Benchmark_Get_IStorageCache_WithoutMetrics_WithPoolAndPreAllocatedKeys(b *testing.B) {
	pk := []byte("pk")
	cc := []byte("cc")
	v := []byte("atestdata")
	var res []byte

	ts := &testStorage{get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
		*data = v
		return true, nil
	}}

	factory := &ByteBufferFactory{}
	pool := go_object_pool.NewFixedPool(testCacheSize, factory)

	//preallocate keys
	oo := make([]go_object_pool.PooledObject, 0)
	for i := 0; i < testCacheSize; i++ {
		o, err := pool.Get()
		if err != nil {
			panic(err)
		}
		oo = append(oo, o)
	}
	for i := range oo {
		err := pool.Return(oo[i])
		if err != nil {
			panic(err)
		}
	}

	storage := &storageCacheWithoutMetricsWithPoolAndPreAllocatedKeys{
		cache:   fastcache.New(testCacheSize),
		pool:    pool,
		storage: ts,
	}

	for i := 0; i < b.N; i++ {
		ok, err := storage.Get(pk, cc, &res)
		if err != nil {
			panic(err)
		}
		if !ok {
			panic("not ok")
		}
		if string(res) != string(v) {
			panic(fmt.Errorf("BUG: invalid value obtained; got %q; want %q", res, v))
		}
	}
}
func Benchmark_Get_IStorageCache_WithoutMetrics_WithPool(b *testing.B) {
	pk := []byte("pk")
	cc := []byte("cc")
	v := []byte("atestdata")
	var res []byte

	ts := &testStorage{get: func(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
		*data = v
		return true, nil
	}}

	factory := &ByteBufferFactory{}
	pool := go_object_pool.NewFixedPool(testCacheSize, factory)

	storage := &storageCacheWithoutMetricsWithPoolAndPreAllocatedKeys{
		cache:   fastcache.New(testCacheSize),
		pool:    pool,
		storage: ts,
	}

	for i := 0; i < b.N; i++ {
		ok, err := storage.Get(pk, cc, &res)
		if err != nil {
			panic(err)
		}
		if !ok {
			panic("not ok")
		}
		if string(res) != string(v) {
			panic(fmt.Errorf("BUG: invalid value obtained; got %q; want %q", res, v))
		}
	}
}

type storageCacheWithoutMetrics struct {
	istorage.IAppStorage
	cache   *fastcache.Cache
	storage istorage.IAppStorage
}

func (s *storageCacheWithoutMetrics) Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	*data = (*data)[0:0]
	*data = s.cache.Get(*data, key(pKey, cCols))
	if len(*data) != 0 {
		return true, err
	}
	ok, err = s.storage.Get(pKey, cCols, data)
	if err != nil {
		return false, err
	}
	if ok {
		s.cache.Set(key(pKey, cCols), *data)
	}
	return
}

type storageCacheWithoutMetricsWithByteBufferPool struct {
	istorage.IAppStorage
	cache   *fastcache.Cache
	storage istorage.IAppStorage
}

func (s *storageCacheWithoutMetricsWithByteBufferPool) Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	bb := bytebufferpool.Get()
	_, err = bb.Write(pKey)
	if err != nil {
		return false, err
	}
	_, err = bb.Write(cCols)
	if err != nil {
		return false, err
	}

	*data = (*data)[0:0]
	*data = s.cache.Get(*data, bb.Bytes())

	bytebufferpool.Put(bb)

	if len(*data) != 0 {
		return true, err
	}
	ok, err = s.storage.Get(pKey, cCols, data)
	if err != nil {
		return false, err
	}
	if ok {
		bb := bytebufferpool.Get()
		_, err = bb.Write(pKey)
		if err != nil {
			return false, err
		}
		_, err = bb.Write(cCols)
		if err != nil {
			return false, err
		}

		s.cache.Set(bb.Bytes(), *data)

		bytebufferpool.Put(bb)
	}
	return
}

type storageCacheWithoutMetricsWithPoolAndPreAllocatedKeys struct {
	istorage.IAppStorage
	cache   *fastcache.Cache
	pool    go_object_pool.Pool
	storage istorage.IAppStorage
}

func (s *storageCacheWithoutMetricsWithPoolAndPreAllocatedKeys) Get(pKey []byte, cCols []byte, data *[]byte) (ok bool, err error) {
	obj, err := s.pool.Get()
	if err != nil {
		return false, err
	}
	bb := obj.(*ByteBufferObject).buffer

	_, err = bb.Write(pKey)
	if err != nil {
		return false, err
	}
	_, err = bb.Write(cCols)
	if err != nil {
		return false, err
	}

	*data = (*data)[0:0]
	*data = s.cache.Get(*data, bb.Bytes())

	err = s.pool.Return(obj)
	if err != nil {
		return false, err
	}

	if len(*data) != 0 {
		return true, err
	}
	ok, err = s.storage.Get(pKey, cCols, data)
	if err != nil {
		return false, err
	}
	if ok {
		obj, err := s.pool.Get()
		if err != nil {
			return false, err
		}
		bb := obj.(*ByteBufferObject).buffer

		_, err = bb.Write(pKey)
		if err != nil {
			return false, err
		}
		_, err = bb.Write(cCols)
		if err != nil {
			return false, err
		}

		s.cache.Set(bb.Bytes(), *data)

		err = s.pool.Return(obj)
		if err != nil {
			return false, err
		}
	}
	return
}

type ByteBufferFactory struct{}

func (f ByteBufferFactory) Create() (go_object_pool.PooledObject, error) {
	return &ByteBufferObject{
		buffer: bytes.NewBuffer(make([]byte, 32)),
	}, nil
}

type ByteBufferObject struct {
	buffer *bytes.Buffer
}

func (b *ByteBufferObject) Reset() {
	b.buffer.Reset()
}
