package response

import (
	//speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

type Response struct {
	TimeStamps		[]int64
	OperationName	string
	Response		*speechpb.LongRunningRecognizeResponse
}