package types

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/zeebo/assert"
)

func TestEndpointMsgpackEncodeDecode(t *testing.T) {

	a1 := &Endpoint{
		ID:             uuid.MustParse("ba36c969-de94-47fd-8c5e-3fc51eead7f4"),
		AppType:        AppTypeEndpoint,
		Name:           "test",
		URL:            "http://localhost:5000/ba36c969-de94-47fd-8c5e-3fc51eead7f4",
		Runtime:        "go",
		ActiveDeployID: uuid.MustParse("5fa1b353-8e3b-43f5-a27c-7260e3d9344e"),
		Environment: map[string]string{
			"TEST": "test",
		},

		// causses difference due to encoding of time.Time
		//CreatedAT: time.Now(),
	}

	b, err := msgpack.Marshal(a1)
	if err != nil {
		t.Fatal(err)
	}

	a2, err := DecodeMsgpakApp(b)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(a1, a2) {
		t.Fatalf("want: %v got: %v", a1, a2)
	}

	assert.Equal(t, "*types.Endpoint", reflect.TypeOf(a2).String())
}

func TestTaskJsonEncodeDecode(t *testing.T) {

	a1 := &Task{
		ID:             uuid.MustParse("ba36c969-de94-47fd-8c5e-3fc51eead7f4"),
		AppType:        AppTypeTask,
		Name:           "test",
		Runtime:        "go",
		ActiveDeployID: uuid.MustParse("5fa1b353-8e3b-43f5-a27c-7260e3d9344e"),
		Environment: map[string]string{
			"TEST": "test",
		},

		// causses difference due to encoding of time.Time
		//CreatedAT: time.Now(),
	}

	b, err := json.Marshal(a1)
	if err != nil {
		t.Fatal(err)
	}

	a2, err := DecodeJsonApp(b)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(a1, a2) {
		t.Fatalf("want: %v got: %v", a1, a2)
	}

	assert.Equal(t, "*types.Task", reflect.TypeOf(a2).String())
}

func TestMsgpackEncodeDecodeUsingDifferentAppType(t *testing.T) {

	// endpoint
	a1 := &Endpoint{
		ID:             uuid.MustParse("ba36c969-de94-47fd-8c5e-3fc51eead7f4"),
		AppType:        AppTypeProcess, // using process type
		Name:           "test",
		Runtime:        "go",
		URL:            "http://localhost:5000/ba36c969-de94-47fd-8c5e-3fc51eead7f4",
		ActiveDeployID: uuid.MustParse("5fa1b353-8e3b-43f5-a27c-7260e3d9344e"),
		Environment: map[string]string{
			"TEST": "test",
		},
	}

	b, err := msgpack.Marshal(a1)
	if err != nil {
		t.Fatal(err)
	}

	a2, err := DecodeMsgpakApp(b)
	if err != nil {
		t.Fatal(err)
	}

	// decode func will return process type
	assert.Equal(t, "*types.Process", reflect.TypeOf(a2).String())
}

func TestJsonEncodeDecodeUsingDifferentAppType(t *testing.T) {

	// endpoint
	a1 := &Endpoint{
		ID:             uuid.MustParse("ba36c969-de94-47fd-8c5e-3fc51eead7f4"),
		AppType:        AppTypeProcess, // using process type
		Name:           "test",
		Runtime:        "go",
		URL:            "http://localhost:5000/ba36c969-de94-47fd-8c5e-3fc51eead7f4",
		ActiveDeployID: uuid.MustParse("5fa1b353-8e3b-43f5-a27c-7260e3d9344e"),
		Environment: map[string]string{
			"TEST": "test",
		},
	}

	b, err := json.Marshal(a1)
	if err != nil {
		t.Fatal(err)
	}

	a2, err := DecodeJsonApp(b)
	if err != nil {
		t.Fatal(err)
	}

	// decode func will return process type
	assert.Equal(t, "*types.Process", reflect.TypeOf(a2).String())
}
