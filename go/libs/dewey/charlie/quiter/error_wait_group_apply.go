package quiter

import (
	"code.linenisgreat.com/chrest/go/libs/dewey/0/interfaces"
	"code.linenisgreat.com/chrest/go/libs/dewey/bravo/errors"
)

func ErrorWaitGroupApply[T any](
	wg errors.WaitGroup,
	s interfaces.Collection[T],
	f interfaces.FuncIter[T],
) bool {
	for e := range s.All() {
		if !wg.Do(
			func() error {
				return f(e)
			},
		) {
			return true
		}
	}

	return false
}
