package javascript

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/ry/v8worker2"
)

var tupeloScript string

func init() {
	file, err := ioutil.ReadFile("./js/dist/index.js")
	if err != nil {
		panic(fmt.Sprintf("error reading standard file: %v", err))
	}
	tupeloScript = string(file)
}

func Run() error {
	worker := v8worker2.New(func(msg []byte) []byte {
		log.Printf("received: %s", string(msg))
		return nil
	})
	err := worker.Load("tupelo.js", tupeloScript)
	if err != nil {
		return fmt.Errorf("error loading file: %v", err)
	}
	return worker.Load("ready.js", `V8Worker2.print("ready");`)
}
