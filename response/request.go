package response

type REQUEST struct {
	URL				string `json:"url" binding:"required"` 
	LOOKING_FOR		string `json:"lookingfor" binding:"required"`
}