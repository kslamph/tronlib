package examplepkg

import "fmt"

// helper is not an example function and must not be extracted.
func helper() {
	fmt.Println("skip me")
}

// ExampleWithParams has parameters, so it is not a doc example and must not
// be extracted even though its name starts with "Example".
func ExampleWithParams(a int) {
	fmt.Println(a)
}

// ExamplePrint carries a comment line that must not leak into the extracted
// body: extracted bodies are comment-free so the // Output: harness line and
// narration never end up inside a rendered doc marker.
func ExamplePrint() {
	// a comment line that must not leak into docs
	fmt.Println("one")
	fmt.Println("two")
}

func ExamplePrint_Two() {
	x := 6
	fmt.Println(x)
}
