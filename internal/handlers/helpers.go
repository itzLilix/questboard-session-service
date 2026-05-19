package handlers

func derefOr[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}
