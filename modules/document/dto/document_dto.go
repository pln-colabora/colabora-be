package dto

import (
	"errors"
	"mime/multipart"
)

const (
	MESSAGE_FAILED_GET_DATA_FROM_BODY = "failed get data from body"
	MESSAGE_FAILED_UPLOAD_DOCUMENT    = "failed upload document"
	MESSAGE_FAILED_GET_DOCUMENT       = "failed get document"
	MESSAGE_FAILED_PREVIEW_DOCUMENT   = "failed preview document"
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
	ErrInvalidFilename         = errors.New("invalid original filename")
	ErrInvalidSupersedesID     = errors.New("supersedes_document_id must be a UUID")
	ErrInvalidRevision         = errors.New("only an unattached upload owned by the caller can be superseded")
	ErrMalwareDetected         = errors.New("malware detected")
	ErrScanFailed              = errors.New("malware scan failed")
	ErrDocumentUnavailable     = errors.New("document is unavailable")
	ErrDocumentAlreadyAttached = errors.New("one or more documents are already attached or do not exist")
)

type (
	DocumentUploadRequest struct {
		File                 *multipart.FileHeader `form:"file" binding:"required"`
		Type                 string                `form:"type" binding:"required"`
		SupersedesDocumentID string                `form:"supersedes_document_id"`
	}

	DocumentResponse struct {
		ID               string   `json:"id"`
		Type             string   `json:"type"`
		OriginalFilename string   `json:"original_filename"`
		MimeType         string   `json:"mime_type"`
		SizeBytes        int64    `json:"size_bytes"`
		ChecksumSHA256   string   `json:"checksum_sha256"`
		Source           string   `json:"source"`
		Classification   string   `json:"classification"`
		ScanStatus       string   `json:"scan_status"`
		ScanCheckedAt    *string  `json:"scan_checked_at"`
		Revision         int16    `json:"revision"`
		SupersedesID     *string  `json:"supersedes_id"`
		SupersededByID   *string  `json:"superseded_by_id"`
		PermohonanID     *string  `json:"permohonan_id"`
		WorkflowNodes    []string `json:"workflow_nodes"`
		UploadedBy       string   `json:"uploaded_by"`
		CreatedAt        string   `json:"created_at"`
	}

	DocumentPreviewResponse struct {
		URL       string `json:"url"`
		MimeType  string `json:"mime_type"`
		Filename  string `json:"filename"`
		ExpiresAt string `json:"expires_at"`
	}
)
