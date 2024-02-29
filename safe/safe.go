package safe

import (
	"errors"
	"fmt"

	"github.com/amh11706/logger"
)

func Go(f func(), onError func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if onError != nil {
					onError()
				}
				message := fmt.Sprintf("Panic recovered: %v", r)
				fmt.Println(message)
				logger.CheckStack(errors.New(message))
			}
		}()

		f()
	}()
}
