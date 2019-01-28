package javascript

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/chaintree/validator/messages"
	"github.com/ry/v8worker2"
)

func init() {
	v8worker2.SetFlags([]string{"--future"})
}

type chanWorker struct {
	worker *v8worker2.Worker
	toJS   chan []byte
	fromJS chan []byte
	stop   chan bool
}

func newChanWorker() *chanWorker {
	fromJS := make(chan []byte)
	worker := v8worker2.New(func(msg []byte) []byte {
		fromJS <- msg
		return nil
	})
	tupeloScript, err := ioutil.ReadFile("./js/dist/index.js")
	if err != nil {
		panic(fmt.Sprintf("error reading standard file: %v", err))
	}

	worker.Load("globals", `const self = {};`)
	err = worker.Load("tupelo", string(tupeloScript))
	if err != nil {
		panic(fmt.Errorf("error loading file: %v", err))
	}
	return &chanWorker{
		worker: worker,
		fromJS: fromJS,
		toJS:   make(chan []byte),
		stop:   make(chan bool),
	}
}

func (cw *chanWorker) Start() {
	for {
		select {
		case <-cw.stop:
			return
		case msg := <-cw.toJS:
			cw.worker.SendBytes(msg)
		}
	}
}

func (cw *chanWorker) Stop() {
	cw.stop <- true
	cw.worker.TerminateExecution()
	close(cw.fromJS)
	close(cw.toJS)
}

func (cw *chanWorker) Load(name, script string) error {
	return cw.worker.Load(name, script)
}

func Validate(tree *chaintree.ChainTree) (result []byte, err error) {
	worker := newChanWorker()
	defer worker.Stop()
	go func() {
		worker.Start()
	}()

	// get the script from the chaintree:
	scriptInt, _, err := tree.Dag.Resolve([]string{"tree", "validIf"})
	if err != nil {
		return nil, fmt.Errorf("no validation script: %v", err)
	}

	err = worker.Load("user-validator.js", scriptInt.(string))
	if err != nil {
		return nil, fmt.Errorf("error loading script: %v", err)
	}

	nodes, err := tree.Dag.Nodes()
	if err != nil {
		return nil, fmt.Errorf("error getting nodes: %v", err)
	}

	bitNodes := make(map[string][]byte)
	for _, n := range nodes {
		bitNodes[n.Cid().String()] = n.RawData()
	}

	sw := safewrap.SafeWrap{}

	start := &messages.Start{
		Tip:   tree.Dag.Tip,
		Nodes: bitNodes,
	}

	any, err := messages.ToAny(start)
	if err != nil {
		return nil, fmt.Errorf("error turning into any: %v", err)
	}

	worker.toJS <- sw.WrapObject(any).RawData()

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil, fmt.Errorf("timeout waiting for result")
	case result := <-worker.fromJS:
		res, err := messages.FromSerialized(result)
		if err != nil {
			return nil, fmt.Errorf("error converting any: %v", err)
		}
		return []byte(res.(*messages.Finished).Result), nil
	}
}

// func Run() error {

// 	worker := v8worker2.New(func(msg []byte) []byte {
// 		log.Printf("received: %s", string(msg))
// 		return nil
// 	})

// 	tupeloScript, err := ioutil.ReadFile("./js/dist/index.js")
// 	if err != nil {
// 		panic(fmt.Sprintf("error reading standard file: %v", err))
// 	}

// 	worker.Load("globals", `const self = {};`)
// 	err = worker.Load("tupelo", string(tupeloScript))
// 	if err != nil {
// 		return fmt.Errorf("error loading file: %v", err)
// 	}

// 	sw := safewrap.SafeWrap{}

// 	obj := map[string]string{"foo": "bar"}

// 	n := sw.WrapObject(obj)

// 	start := &messages.Start{
// 		Tip:   n.Cid(),
// 		Nodes: [][]byte{n.RawData()},
// 	}

// 	any, err := messages.ToAny(start)
// 	if err != nil {
// 		return fmt.Errorf("error turning into any: %v", err)
// 	}

// 	data := sw.WrapObject(any).RawData()
// 	log.Printf("sending: %s", base64.StdEncoding.EncodeToString(data))
// 	err = worker.SendBytes(data)
// 	if err != nil {
// 		return fmt.Errorf("error sending bytes: %v", err)
// 	}
// 	return worker.Load("ready.js", `V8Worker2.print("ready");`)
// }
