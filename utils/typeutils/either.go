package typeutils

type Either[L any, R any] struct {
	left  Optional[L]
	right Optional[R]
}

func Left[L any, R any](value L) Either[L, R] {
	return Either[L, R]{left: Some(value)}
}

func Right[L any, R any](value R) Either[L, R] {
	return Either[L, R]{right: Some(value)}
}

func (e Either[L, R]) IsLeft() bool {
	return e.left.IsSome()
}

func (e Either[L, R]) IsRight() bool {
	return e.right.IsSome()
}

func (e Either[L, R]) UnwrapLeft() (L, error) {
	return e.left.Unwrap()
}

func (e Either[L, R]) UnwrapRight() (R, error) {
	return e.right.Unwrap()
}

func (e Either[L, R]) ForceUnwrapLeft() L {
	return e.left.ForceUnwrap()
}

func (e Either[L, R]) ForceUnwrapRight() R {
	return e.right.ForceUnwrap()
}
