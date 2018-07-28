package dag

import (
	"testing"
	"math"
	"time"
	//realcbor "github.com/ipfs/go-ipld-cbor"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"github.com/quorumcontrol/chaintree/cbornode"
	//"github.com/polydawn/refmt/cbor"
)

type testEncodeStruct struct {
	Id string
	Number int
}

func TestParallelWrap(t *testing.T) {
	cbornode.RegisterCborType(testEncodeStruct{})

	times := 10000

	obj := &testEncodeStruct{Id: "test", Number: 100}

	responses := make(chan int, times)

	for i := 0; i < times; i++ {
		go func() {
			now := time.Now()
			node,err := cbornode.WrapObject(obj, multihash.SHA2_256, -1)
			end := int(time.Now().Sub(now))
			assert.Nil(t, err)
			assert.Len(t, node.RawData(), 18)
			responses <- end
		}()
	}

	respSlice := make([]int, times)

	for i := 0; i < times; i++ {
		respSlice[i] = <-responses
	}
	t.Logf("max was: %v", max(respSlice))

	assert.True(t, max(respSlice)< int(100 * time.Millisecond))

}


func max(ints []int) int {
	max := 0
	for i := 0; i < len(ints); i++ {
		if ints[i] > max {
			max = ints[i]
		}
	}
	return max
}

func min(ints []int) int {
	min := math.MaxInt64
	for i := 0; i < len(ints); i++ {
		if ints[i] < min {
			min = ints[i]
		}
	}
	return min
}
