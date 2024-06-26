package lingo

import (
	"sync"

	"github.com/vanh01/lingo/definition"
)

// GroupBy groups elements that share a common attribute.
//
// In this method, getHash returns the key of the grouping.
//
// elementSelector, getHash can be nil
func (e Enumerable[T]) GroupBy(
	keySelector definition.SingleSelector[T],
	elementSelector definition.SingleSelector[T],
	resultSelector definition.GroupBySelector[any, any],
	getHash definition.GetHashCode[any],
) Enumerable[any] {
	return Enumerable[any]{
		getIter: func() <-chan any {
			out := make(chan any)

			go func() {
				defer close(out)
				res := map[any][]any{}
				for value := range e.getIter() {
					var element any
					key := keySelector(value)
					if elementSelector != nil {
						element = elementSelector(value)
					} else {
						element = value
					}
					if getHash != nil {
						key = getHash(key)
					}
					res[key] = append(res[key], element)
				}
				for k, v := range res {
					out <- resultSelector(k, v)
				}
			}()

			return out
		},
	}
}

// ParallelEnumerable

// GroupBy groups in parallel elements that share a common attribute.
//
// In this method, getHash returns the key of the grouping.
//
// elementSelector, getHash can be nil
func (p ParallelEnumerable[T]) GroupBy(
	keySelector definition.SingleSelector[T],
	elementSelector definition.SingleSelector[T],
	resultSelector definition.GroupBySelector[any, any],
	getHash definition.GetHashCode[any],
) ParallelEnumerable[any] {
	return ParallelEnumerable[any]{
		getIter: func() <-chan any {
			mapdata := make(chan definition.KeyValData[any, any])

			res := map[any][]any{}

			go func() {
				defer close(mapdata)
				var wg sync.WaitGroup
				for value := range p.getIter() {
					wg.Add(1)
					temp := value
					go func() {
						defer wg.Done()

						ele := make(chan any)
						go func() {
							defer close(ele)
							var element any = temp
							if elementSelector != nil {
								element = elementSelector(temp)
							}
							ele <- element
						}()

						key := keySelector(temp)
						if getHash != nil {
							key = getHash(key)
						}

						mapdata <- definition.KeyValData[any, any]{
							Key: key,
							Val: <-ele,
						}
					}()
				}
				wg.Wait()
			}()

			for d := range mapdata {
				res[d.Key] = append(res[d.Key], d.Val)
			}

			out := make(chan any)

			go func() {
				defer close(out)
				var wg sync.WaitGroup
				for k, v := range res {
					wg.Add(1)
					k1, v1 := k, v
					go func() {
						defer wg.Done()
						out <- resultSelector(k1, v1)
					}()
				}
				wg.Wait()
			}()

			return out
		},
	}
}
