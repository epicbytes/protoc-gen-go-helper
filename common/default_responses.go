package common

// swagger:response CommentedResponse
// CommentedResponseWrapper ...
type CommentedResponseWrapper struct {
	// in:body
	Body *CommentedResponse
}

// swagger:response FileResponse
// FileResponseWrapper ...
type FileResponseWrapper struct {
	// in:body
	Body *FileResponse
}

// swagger:response ErrorResponse
type GeneralErrorWrapper struct {
	// in: body
	Body interface{}
}

// swagger:response ErrorValidation
type GeneralValidationErrorWrapper struct {
	// in: body
	Body interface{}
}
