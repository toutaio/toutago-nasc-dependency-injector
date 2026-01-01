package nasc

import "fmt"

// Example types for documentation
type ExampleGreeter interface {
	Greet() string
}

type ExampleSimpleGreeter struct{}

func (g *ExampleSimpleGreeter) Greet() string {
	return "Hello, Nasc!"
}

func ExampleNew() {
	container := New()
	fmt.Printf("Container created: %v\n", container != nil)
	// Output: Container created: true
}

func ExampleNasc_Bind() {
	container := New()

	// Bind an interface to an implementation
	err := container.Bind((*Logger)(nil), &ConsoleLogger{})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Binding successful")
	// Output: Binding successful
}

func ExampleNasc_Make() {
	container := New()

	// Bind and resolve
	_ = container.Bind((*ExampleGreeter)(nil), &ExampleSimpleGreeter{})
	instance := container.Make((*ExampleGreeter)(nil))

	greeter := instance.(ExampleGreeter)
	fmt.Println(greeter.Greet())
	// Output: Hello, Nasc!
}
