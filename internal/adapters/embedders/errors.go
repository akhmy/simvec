package embedders

import "fmt"

type UnexpectedResponseStatusError struct {
	Has  int
	Want int
}

func (e *UnexpectedResponseStatusError) Error() string {
	return fmt.Sprintf("unexpected response status code, has '%d', want '%d'", e.Has, e.Want)
}
