package safe

import (
	"errors"
	"fmt"
	"log"

	"github.com/amh11706/logger"
)

func doRecover(onError func(interface{})) {
	if r := recover(); r != nil {
		if onError != nil {
			onError(r)
		}
		message := fmt.Sprintf("Panic recovered: %v", r)
		log.Println(message)
		logger.CheckStack(errors.New(message))
	}
}

func Go(f func(), onError func(interface{})) {
	go func() {
		defer doRecover(onError)
		f()
	}()
}

func GoWithValue[T any](f func(T), value T, onError func(interface{})) {
	go func() {
		defer doRecover(onError)
		f(value)
	}()
}
