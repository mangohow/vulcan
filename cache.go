package vulcan

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"
)

/*
*
如果是local cache, 则需要在实现CacheManger的Get和Set内部处理数据拷贝, 防止外部修改了缓存中的数据
*/
type CacheManger[T any] interface {
	Get(key string) (*T, bool)
	Set(key string, value *T)
	Delete(key string)
}

type cacheKey struct{}

type CacheConfig[T any] struct {
	Manager          CacheManger[T]
	Key              string
	CacheNil         bool
	QueryTimeOut     time.Duration
	BeforeInvocation bool
	flightGroup      singleflight.Group
}

func getCacheInterceptor(ctx context.Context) InterceptorHandler {
	if ctx == nil {
		return nil
	}

	value := ctx.Value(cacheKey{})
	if value == nil {
		return nil
	}

	return value.(InterceptorHandler)
}

func CacheableCtx[T any](cfg *CacheConfig[T]) context.Context {
	return context.WithValue(context.Background(), cacheKey{}, InterceptorHandler(func(option *ExecOption, next Handler) (any, error) {
		return cacheableHandler(cfg, option, next)
	}))
}

func CacheEvictCtx[T any](cfg *CacheConfig[T]) context.Context {
	return context.WithValue(context.Background(), cacheKey{}, InterceptorHandler(func(option *ExecOption, next Handler) (any, error) {
		return cacheEvictInterceptor(cfg, option, next)
	}))
}

type result struct {
	data any
	err  error
}

func cacheableHandler[T any](cfg *CacheConfig[T], option *ExecOption, next Handler) (*T, error) {
	if cfg.Key == "" {
		return nil, fmt.Errorf("empty key provided")
	}

	// 1、查询缓存
	if val, exist := cfg.Manager.Get(cfg.Key); exist {
		return val, nil
	}

	v, err, _ := cfg.flightGroup.Do(cfg.Key, func() (any, error) {
		ctx := context.Background()
		if cfg.QueryTimeOut > 0 {
			c, cancel := context.WithTimeout(ctx, cfg.QueryTimeOut)
			ctx = c
			defer cancel()
		}
		resCh := make(chan result)

		go func() {
			// 2、缓存不存在, 查询数据库
			val, err := next(option)
			if err != nil {
				resCh <- result{nil, err}
				return
			}

			var (
				objPtr *T
				obj    T
				ok     bool
			)

			if val == nil {
				if cfg.CacheNil {
					objPtr = nil
				} else {
					resCh <- result{nil, nil}
				}
			} else {
				objPtr, ok = val.(*T)
				if !ok {
					obj = val.(T)
					objPtr = &obj
				}
			}

			// 3、写入缓存
			cfg.Manager.Set(cfg.Key, objPtr)
			resCh <- result{objPtr, nil}
		}()

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("query db timeout, key: %s", cfg.Key)
		case res := <-resCh:
			return res.data, res.err
		}
	})

	if err != nil {
		return nil, err
	}

	return v.(*T), nil
}

func cacheEvictInterceptor[T any](cfg *CacheConfig[T], option *ExecOption, next Handler) (any, error) {
	// 先删缓存
	if cfg.BeforeInvocation {
		cfg.Manager.Delete(cfg.Key)
	}

	// 再更新数据
	res, err := next(option)
	if err != nil {
		return nil, err
	}

	if !cfg.BeforeInvocation {
		cfg.Manager.Delete(cfg.Key)
	}

	return res, nil
}
