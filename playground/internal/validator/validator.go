package validator

// Validator serves as the centralized structure for tracking payload errors
type Validator struct {
	Errors map[string]string
}

// New instantiates a fresh validation container
func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

// HasErrors returns true if any issues are currently logged
func (v *Validator) HasErrors() bool {
	return len(v.Errors) > 0
}

// AddError logs a distinct validation error message if the field isn't already tracked
func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

// Check adds an error message to the log only if an evaluation rule evaluates to false
func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}
