package response

type REQUEST struct {
	WORKER_ID		string `json:"workerid" binding:"required"`
}