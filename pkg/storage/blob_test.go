package storage

import (
	"testing"
)

func TestGetPath(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"7ed91c53-ef10-4b92-b059-ba0c1cc7eadd", "./7/7e/7ed/7ed91c53-ef10-4b92-b059-ba0c1cc7eadd.bin"},
		{"01234", "./0/01/012/01234.bin"},
	}

	store := NewDiskBlobStore(BlobStoreConfig{
		BaseDir: ".",
		Host:    false,
	})
	for _, tc := range testCases {
		actual := store.GetPath(tc.input)
		if actual != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, actual)
		}
	}

}
