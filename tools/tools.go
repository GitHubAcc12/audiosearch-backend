package tools

import (
	"os"
	"os/exec"
	"strconv"
	"encoding/json"
	"log"

	"backend/response"
	"backend/constants"
)

func ToSlice(c chan response.Response) []response.Response {
	s := make([]response.Response, 0)
	for i := range c {
		s = append(s, i)
	}
	return s
}

func WorkerDoesntExistResponse() string {
	resp := response.Response{
		TimeStamps: []int64{},
		Message: "Unknown WorkerID submitted",
		GoogleResponse: nil,
		Index: int64(-1),
		WorkerId: "", // Getting there
	}
	jsonResult, err := json.Marshal(resp)

	if err != nil {
		log.Fatal(err)
	}

	return string(jsonResult)
}

func RemoveContents(dir string) error {
	return os.RemoveAll(dir)
}


func GetCommandAudioFromVideofile(inputFile string, outputFile string) *exec.Cmd{
	// ffmpeg -i {video_name} -ab 160k -ac 2 -ar 44100 -vn audio.wav
	return exec.Command("ffmpeg",
	"-i", inputFile,
	"-ab", "160k",
	"-ac", "2",
	"-ar", "44100",
	"-vn", outputFile)
}

func GetCommandSplitAudio(inputFile string, outputPath string) *exec.Cmd {
	// ffmpeg -i precalcday1.wav -f segment -segment_time 600 -c copy out%03d.wav
	// Segment audio file into 5 minute long pieces
	return  exec.Command("ffmpeg",
	"-i", inputFile,
	"-f", "segment",
	"-segment_time", strconv.Itoa(constants.AUDIO_SEGMENT_LENGTH_SECONDS),
	"-c", "copy",
	outputPath+"%03d.wav")
}