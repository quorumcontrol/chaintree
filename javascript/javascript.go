package javascript

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/quorumcontrol/chaintree/javascript/messages"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/ry/v8worker2"
)

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
	v8worker2.SetFlags([]string{"--future", "true"})

	worker := v8worker2.New(func(msg []byte) []byte {
		log.Printf("received: %s", string(msg))
		return nil
	})

	tupeloScript, err := ioutil.ReadFile("./js/dist/index.js")
	if err != nil {
		panic(fmt.Sprintf("error reading standard file: %v", err))
	}

	worker.Load("globals", `const self = {};`)
	err = worker.Load("tupelo", string(tupeloScript))
	if err != nil {
		return fmt.Errorf("error loading file: %v", err)
	}

	sw := safewrap.SafeWrap{}

	obj := map[string]string{"foo": "bar"}

	n := sw.WrapObject(obj)

	start := &messages.Start{
		Tip:   n.Cid(),
		Nodes: [][]byte{n.RawData()},
	}

	any, err := messages.ToAny(start)
	if err != nil {
		return fmt.Errorf("error turning into any: %v", err)
	}

	data := sw.WrapObject(any).RawData()
	log.Printf("sending: %s", base64.StdEncoding.EncodeToString(data))
	err = worker.SendBytes(data)
	if err != nil {
		return fmt.Errorf("error sending bytes: %v", err)
	}
	return worker.Load("ready.js", `V8Worker2.print("ready");`)
}
