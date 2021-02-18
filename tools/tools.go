package tools

import(
	"log"

	"backend/response"
)

func ToSlice(c chan response.Response) []response.Response {
	s := make([]response.Response, 0)
	for i := range c {
		log.Print("iterating through channel")
		s = append(s, i)
	}
	return s
}