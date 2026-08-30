package key

import (
	"fmt"
	"testing"
)

func TestGenerate(t *testing.T) {
	plain, hash, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(plain)
	fmt.Println(hash)
}
