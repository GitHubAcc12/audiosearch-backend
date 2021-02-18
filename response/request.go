package response

type REQUEST struct {
	URI				string `json:"uri" binding:"required"` 
	LOOKING_FOR		string `json:"lookingfor" binding:"required"`
	WORKER_ID		string `json:"workerid" binding:"required"`
}