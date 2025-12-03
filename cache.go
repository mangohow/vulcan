package vulcan

import (
	"context"
)

type Cacheable interface {
	CacheKey() string
}

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
	BeforeInvocation bool
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

func cacheableHandler[T any](cfg *CacheConfig[T], option *ExecOption, next Handler) (*T, error) {
	// 1、查询缓存
	if val, exist := cfg.Manager.Get(cfg.Key); exist {
		// 拷贝一份
		res := *val
		return &res, nil
	}

	// 2、缓存不存在, 查询数据库
	val, err := next(option)
	if err != nil {
		return nil, err
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
			return nil, nil
		}
	} else {
		objPtr, ok = val.(*T)
		if !ok {
			obj = val.(T)
			objPtr = &obj
		}
	}

	// 3、写入缓存
	if objPtr == nil {
		cfg.Manager.Set(cfg.Key, nil)
	} else {
		cachedObj := *objPtr
		cfg.Manager.Set(cfg.Key, &cachedObj)
	}

	return objPtr, nil
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
