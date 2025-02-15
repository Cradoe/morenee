package validator

type Validator struct {
	Errors []string `json:",omitempty"`
}

func (v Validator) HasErrors() bool {
	return len(v.Errors) != 0
}

func (v *Validator) AddError(message string) {
	if v.Errors == nil {
		v.Errors = []string{}
	}

	v.Errors = append(v.Errors, message)
}

func (v *Validator) Check(ok bool, message string) {
	if !ok {
		v.AddError(message)
	}
}
