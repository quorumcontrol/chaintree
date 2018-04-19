package typecaster

import (
	"github.com/polydawn/refmt/obj/atlas"
	"github.com/polydawn/refmt/obj"
	"github.com/polydawn/refmt/shared"
	"sync"
)

var currentAtlas atlas.Atlas
var atlasMutex sync.Mutex
var entries []*atlas.AtlasEntry

func AddType(typeHint interface{}) {
	atlasMutex.Lock()
	defer atlasMutex.Unlock()
	entry := atlas.BuildEntry(typeHint).StructMap().Autogenerate().Complete()
	entries = append(entries, entry)
	currentAtlas = atlas.MustBuild(entries...)
}


func ToType(src, dst interface{}) error {
	return ToTypeAtlasted(src, dst, currentAtlas)
}

func ToTypeAtlasted(src, dst interface{}, atl atlas.Atlas) error {
	return NewTyper(atl).ToType(src, dst)
}

type Typer interface {
	ToType(src, dst interface{}) error
}

func NewTyper(atl atlas.Atlas) Typer {
	x := &typer{
		marshaller:   obj.NewMarshaller(atl),
		unmarshaller: obj.NewUnmarshaller(atl),
	}
	x.pump = shared.TokenPump{x.marshaller, x.unmarshaller}
	return x
}

type typer struct {
	marshaller   *obj.Marshaller
	unmarshaller *obj.Unmarshaller
	pump         shared.TokenPump
}

func (c typer) ToType(src, dst interface{}) error {
	c.marshaller.Bind(src)
	c.unmarshaller.Bind(dst)
	return c.pump.Run()
}