package storage

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestGetObjectReturnsStorageBody(t *testing.T) {
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, request.Method)
		require.Equal(t, "/documents/documents/test.pdf", request.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/pdf"}},
			Body:       io.NopCloser(strings.NewReader("private document")),
			Request:    request,
		}, nil
	})

	client := s3.New(s3.Options{
		Region:       "garage",
		BaseEndpoint: aws.String("http://garage.test"),
		UsePathStyle: true,
		HTTPClient:   &http.Client{Transport: transport},
		Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: "test", SecretAccessKey: "test"}, nil
		}),
	})

	body, err := NewS3Client(client, "documents").GetObject(context.Background(), "documents/test.pdf")
	require.NoError(t, err)
	defer body.Close()
	content, err := io.ReadAll(body)
	require.NoError(t, err)
	require.Equal(t, "private document", string(content))
}
