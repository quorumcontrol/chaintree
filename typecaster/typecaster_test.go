package typecaster

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

type transaction struct {
	Type string `refmt:"type"`
	Payload interface{}
}

func init() {
	AddType(transaction{})
}

func TestToType(t *testing.T) {
	trans := &transaction{
		Type: "ADD_DATA",
		Payload: map[string]interface{}	{
			"path": "child/is/good",
			"value": "good",
		},
	}


	jsonish := map[string]interface{} {
		"type": "ADD_DATA",
		"payload": map[string]string {
			"path": "child/is/good",
			"value": "good",
		},
	}

	newTrans := &transaction{}

	err := ToType(jsonish, newTrans)

	assert.Nil(t, err)

	assert.Equal(t, trans, newTrans)
}
