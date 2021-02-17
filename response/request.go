package response

type REQUEST struct {
	URI				string `json:"url" binding:"required"` 
	LOOKING_FOR		string `json:"lookingfor" binding:"required"`
}