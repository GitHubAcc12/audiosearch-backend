package response

import (
	"backend/constants"

	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

type Response struct {
	TimeStamps		[]int64
	Message			string
	GoogleResponse	*speechpb.RecognizeResponse
	Index			int64
	WorkerId		string
}


func (r *Response) FindWordTimestamps(word string) {
	r.TimeStamps = make([]int64, 0)
	if len(r.GoogleResponse.Results) == 0 {
		return
	}
	for _, foundWord := range r.GoogleResponse.Results[0].Alternatives[0].Words {
		if foundWord.Word == word {
			r.TimeStamps = append(r.TimeStamps, foundWord.StartTime.GetSeconds()+r.Index*constants.AUDIO_SEGMENT_LENGTH_SECONDS)
		}
	}
}