package serialization

import (
	"github.com/polydawn/refmt/obj/atlas"
	"testing"
	"time"
	"github.com/polydawn/refmt/cbor"
	"github.com/stretchr/testify/assert"
	"github.com/ugorji/go/codec"
)

var cborAtlas atlas.Atlas
//var atlasEntries = []*atlas.AtlasEntry{cidAtlasEntry, bigIntAtlasEntry}
var atlasEntries = make([]*atlas.AtlasEntry,0)

func init() {
	cborAtlas = atlas.MustBuild()
	RegisterCborType(testEncodeStruct{})
}

// RegisterCborType allows to register a custom cbor type
func RegisterCborType(i interface{}) {
	var entry *atlas.AtlasEntry
	if ae, ok := i.(*atlas.AtlasEntry); ok {
		entry = ae
	} else {
		entry = atlas.BuildEntry(i).StructMap().Autogenerate().Complete()
	}
	atlasEntries = append(atlasEntries, entry)
	cborAtlas = atlas.MustBuild(atlasEntries...)
}


type testEncodeStruct struct {
	Id string
	Number int
}

func TestPlay(t *testing.T) {
	obj := &testEncodeStruct{Id: "test", Number: 100}

	var b []byte = make([]byte, 0)
	cborHandle := new(codec.CborHandle)
	cborHandle.Canonical = true
	var h codec.Handle = cborHandle
	var enc *codec.Encoder = codec.NewEncoderBytes(&b, h)
	var err error = enc.Encode(obj)
	assert.Nil(t,err)

	dec := codec.NewDecoderBytes(b, cborHandle)

	var m interface{}
	err = dec.Decode(&m)
	assert.Nil(t, err)

	assert.Len(t, b, 18)
}

func TestParallelCodec(t *testing.T) {
	obj := &testEncodeStruct{Id: "test", Number: 100}

	cborHandle := new(codec.CborHandle)
	cborHandle.Canonical = true
	var h codec.Handle = cborHandle

	times := 10000

	responses := make(chan int, times)

	for i := 0; i < times; i++ {
		go func() {
			b := make([]byte, 0)
			var m interface{}

			now := time.Now()

			var enc *codec.Encoder = codec.NewEncoderBytes(&b, h)
			var err error = enc.Encode(obj)
			dec := codec.NewDecoderBytes(b, cborHandle)
			err = dec.Decode(&m)

			end := int(time.Now().Sub(now))

			assert.Nil(t, err)
			assert.Len(t, b, 18)
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

func TestParallelWrap(t *testing.T) {

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


func TestParallelWrap2(t *testing.T) {

	times := 10000

	obj := &testEncodeStruct{Id: "test", Number: 100}

	responses := make(chan time.Duration, times)

	launchStart := time.Now()
	for i := 0; i < times; i++ {
		go func() {
			var m interface{}
			now := time.Now()
			data, err := cbor.MarshalAtlased(obj, cborAtlas)
			if err != nil {
				panic("error marshaling")
			}

			cbor.UnmarshalAtlased(data, &m, cborAtlas)

			end := time.Now().Sub(now)
			responses <- end
		}()
	}
	t.Logf("launch fanout took: %v", time.Now().Sub(launchStart))

	respSlice := make([]time.Duration, times)

	for i := 0; i < times; i++ {
		respSlice[i] = <-responses
	}
	t.Logf("max was: %v", maxTime(respSlice))
	t.Fail()
}


func maxTime(ints []time.Duration) time.Duration {
	max := time.Duration(0)
	for i := 0; i < len(ints); i++ {
		if ints[i] > max {
			max = ints[i]
		}
	}
	return max
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
