package flux

type Validator interface {
	Validate(i interface{}) error
}
