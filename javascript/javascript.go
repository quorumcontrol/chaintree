package javascript

import (
	"github.com/ry/v8worker2"
)

func Run() error {
	worker := v8worker2.New(func(msg []byte) []byte {
		panic("should not receive err")
		return nil
	})
	worker.SendBytes()
	return worker.Load("code.js", `V8Worker2.print("ready");`)
}
