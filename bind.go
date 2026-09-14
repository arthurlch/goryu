package goryu

// Simple binder, I will find it a better place later on !

func Bind[T any](c *Ctx) (T, error) {
	var v T
	err := c.BodyParser(&v)
	return v, err
}
