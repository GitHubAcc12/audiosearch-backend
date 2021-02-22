package tools

import(
	"os/exec"
	"strconv"
	"log"

	"backend/response"
	"backend/constants"
)

func ToSlice(c chan response.Response) []response.Response {
	log.Print("In toslice")
	s := make([]response.Response, 0)
	for i := range c {
		if len(i.GoogleResponse.Results) > 0 {
			log.Print(i.GoogleResponse.Results[0].Alternatives[0].Transcript)
		}
		
		s = append(s, i)
	}
	log.Print("Toslice triple safe")
	for _, r := range s {
		if len(r.GoogleResponse.Results) > 0 {
			log.Print(r.GoogleResponse.Results[0].Alternatives[0].Transcript)
		}
		
	}
	return s
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