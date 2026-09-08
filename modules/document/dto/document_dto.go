package dto

import (
	"errors"
	"mime/multipart"
)

const (
	MESSAGE_FAILED_GET_DATA_FROM_BODY = "failed get data from body"
	MESSAGE_FAILED_UPLOAD_DOCUMENT    = "failed upload document"
	MESSAGE_FAILED_GET_DOCUMENT       = "failed get document"
	MESSAGE_FAILED_GET_LIST_DOCUMENT  = "failed get list document"

	MESSAGE_SUCCESS_UPLOAD_DOCUMENT   = "success upload document"
	MESSAGE_SUCCESS_GET_DOCUMENT      = "success get document"
	MESSAGE_SUCCESS_GET_LIST_DOCUMENT = "success get list document"
)

var (
	ErrPermohonanNotFound      = errors.New("permohonan not found")
	ErrDocumentNotFound        = errors.New("document not found")
	ErrNotWorkflowNodeOwner    = errors.New("user does not own this workflow node")
	ErrWorkflowNodeNotFound    = errors.New("workflow node not found")
	ErrInvalidFileType         = errors.New("file type not allowed")
	ErrFileTooLarge            = errors.New("file exceeds max upload size")
	ErrInvalidDocumentType     = errors.New("type is required")
	ErrDocumentAlreadyAttached = errors.New("one or more documents are already attached or do not exist")
)

type (
	DocumentUploadRequest struct {
		File *multipart.FileHeader `form:"file" binding:"required"`
		Type string                `form:"type" binding:"required"`
	}

	DocumentResponse struct {
		ID            string   `json:"id"`
		Type          string   `json:"type"`
		PermohonanID  *string  `json:"permohonan_id"`
		WorkflowNodes []string `json:"workflow_nodes"`
		UploadedBy    string   `json:"uploaded_by"`
		CreatedAt     string   `json:"created_at"`
	}
)
