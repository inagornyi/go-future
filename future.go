package future

import (
	"fmt"
	"sync"
)

type tuple[T1, T2 any] struct {
	_1 T1
	_2 T2
}

type Future[T any] struct {
	val T
	err error

	pending bool
	mutex   *sync.Mutex
	wg      *sync.WaitGroup
}

func New[T any](executor func(resolve func(T), reject func(error))) *Future[T] {
	if executor == nil {
		panic("computation cannot be nil")
	}

	f := &Future[T]{
		pending: true,
		mutex:   &sync.Mutex{},
		wg:      &sync.WaitGroup{},
	}

	f.wg.Add(1)

	go func() {
		defer f.handlePanic()
		executor(f.resolve, f.reject)
	}()

	return f
}

func (f *Future[T]) resolve(val T) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if !f.pending {
		return
	}

	f.val = val
	f.pending = false

	f.wg.Done()
}

func (f *Future[T]) reject(err error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if !f.pending {
		return
	}

	f.err = err
	f.pending = false

	f.wg.Done()
}

func (f *Future[T]) handlePanic() {
	err := recover()
	if validErr, ok := err.(error); ok {
		f.reject(fmt.Errorf("panic recovery: %w", validErr))
	} else {
		f.reject(fmt.Errorf("panic recovery: %+v", err))
	}
}

func Then[A, B any](future *Future[A], resolveA func(data A) B) *Future[B] {
	return New(func(resolveB func(B), reject func(error)) {
		result, err := future.Await()
		if err != nil {
			reject(err)
			return
		}
		resolveB(resolveA(result))
	})
}

func Catch[T any](future *Future[T], rejection func(err error) error) *Future[T] {
	return New(func(resolve func(T), reject func(error)) {
		result, err := future.Await()
		if err != nil {
			reject(rejection(err))
			return
		}
		resolve(result)
	})
}

func Value[T any](val T) *Future[T] {
	return &Future[T]{
		val:     val,
		pending: false,
		mutex:   &sync.Mutex{},
		wg:      &sync.WaitGroup{},
	}
}

func Error[T any](err error) *Future[T] {
	return &Future[T]{
		err:     err,
		pending: false,
		mutex:   &sync.Mutex{},
		wg:      &sync.WaitGroup{},
	}
}

func (f *Future[T]) Await() (T, error) {
	f.wg.Wait()
	return f.val, f.err
}

func Wait[T any](futures ...*Future[T]) *Future[[]T] {
	if len(futures) == 0 {
		return nil
	}

	return New(func(resolve func([]T), reject func(error)) {
		valsChan := make(chan tuple[T, int], len(futures))
		errsChan := make(chan error, 1)

		for idx, f := range futures {
			idx := idx
			_ = Then(f, func(data T) T {
				valsChan <- tuple[T, int]{_1: data, _2: idx}
				return data
			})
			_ = Catch(f, func(err error) error {
				errsChan <- err
				return err
			})
		}

		resolutions := make([]T, len(futures))
		for idx := 0; idx < len(futures); idx++ {
			select {
			case val := <-valsChan:
				resolutions[val._2] = val._1
			case err := <-errsChan:
				reject(err)
				return
			}
		}
		resolve(resolutions)
	})
}
