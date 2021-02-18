package response

import (
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

type Response struct {
	TimeStamps		[]int64
	Message			string
	Response		*speechpb.RecognizeResponse
	Index			int
	WorkerId		string
}