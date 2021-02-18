package tools

import(
	"log"

	"backend/response"
)

func ToSlice(c chan response.Response) []response.Response {
	log.Print("In ToSlice, len c: " + string(len(c)))
	s := make([]response.Response, 0)
	for i := range c {
		log.Print("iterating through channel")
		s = append(s, i)
	}
	log.Print(s)
	return s
}