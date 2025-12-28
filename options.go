package nasc

// Option is a function that configures a Nasc container.
type Option func(*Nasc) error

// WithDebug enables debug mode for the container.
// Phase 1: This is a placeholder for future implementation.
func WithDebug() Option {
	return func(n *Nasc) error {
		// TODO: Implement debug mode in Phase 7
		return nil
	}
}

// WithValidation enables strict validation mode.
// Phase 1: This is a placeholder for future implementation.
func WithValidation() Option {
	return func(n *Nasc) error {
		// TODO: Implement validation mode in Phase 7
		return nil
	}
}
