package ptr

// Val 指针安全解引用，nil返回类型零值，全类型通用
func Val[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// Ptr 普通值转指针（更新DTO赋值常用）
func Ptr[T any](v T) *T {
	return &v
}
