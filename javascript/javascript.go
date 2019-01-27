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

type modLoader struct {
	worker *v8worker2.Worker
}

func (ml *modLoader) load(mod, referrer string) int {
	// log.Printf("mod: %s, ref: %s", mod, referrer)
	// if strings.HasPrefix(referrer, "tupelo") {
	// 	file, err := ioutil.ReadFile("./js/" + strings.TrimPrefix(mod, "./") + ".js")
	// 	if err != nil {
	// 		log.Printf("error getting file: %v", err)
	// 		return 1
	// 	}
	// 	err = ml.worker.LoadModule("tupelo/"+mod, string(file), ml.load)
	// 	if err != nil {
	// 		log.Printf("error loading module: %v", err)
	// 		return 1
	// 	}
	// 	return 0
	// }
	return 1
}

func Run() error {
	worker := v8worker2.New(func(msg []byte) []byte {
		log.Printf("received: %s", string(msg))
		return nil
	})

	worker.Load("globals", `const self = {};`)
	err := worker.Load("tupelo", tupeloScript)
	if err != nil {
		return fmt.Errorf("error loading file: %v", err)
	}
	return worker.Load("ready.js", `V8Worker2.print("ready");`)
}
