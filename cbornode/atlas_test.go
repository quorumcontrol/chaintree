package cbornode

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"github.com/polydawn/refmt/cbor"
)

type testEncodeStruct struct {
	Id string
	Number int
}

func TestParallelWrap(t *testing.T) {
	RegisterCborType(testEncodeStruct{})

	times := 10000

	obj := &testEncodeStruct{Id: "test", Number: 100}

	responses := make(chan int, times)

	for i := 0; i < times; i++ {
		go func() {
			var m interface{}
			now := time.Now()
			data, err := cbor.MarshalAtlased(obj, cborAtlas)
			if err != nil {
				panic("error marshaling")
			}

			cbor.UnmarshalAtlased(data, &m, cborAtlas)

			end := int(time.Now().Sub(now))
			assert.Nil(t, err)
			assert.Len(t, data, 18)
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