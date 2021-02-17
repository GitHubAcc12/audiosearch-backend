package tools

func ToSlice(c chan string) []string {
	s := make([]string, 0)
	for i := range c {
		s = append(s, i)
	}
	return s
}