package safe

import (
	"fmt"
	"runtime/debug"
)

func Go(f func(), onError func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if onError != nil {
					onError()
				}
				fmt.Println("WS Panic recovered:", r)
				debug.PrintStack()
			}
		}()

		f()
	}()
}
