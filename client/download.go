package client

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

func downloadDocument(ctx context.Context, api *tg.Client, doc *tg.InputDocument) (string, error) {
	location := &tg.InputDocumentFileLocation{
		ID:            doc.ID,
		AccessHash:    doc.AccessHash,
		FileReference: doc.FileReference,
	}

	// future proofing - for this application all download documents will be rather small.
	var content []byte
	var offset int64 = 0
	const limit = 1024 * 1024 // 1MB chunks

	for {
		result, err := api.UploadGetFile(ctx, &tg.UploadGetFileRequest{
			Location: location,
			Offset:   offset,
			Limit:    limit,
		})

		if err != nil {
			return "", &MessageError{Err: fmt.Errorf("failed to download file: %w", err)}
		}

		bytes, ok := result.(*tg.UploadFile)
		if !ok {
			return "", &MessageError{Err: fmt.Errorf("unexpected result type")}
		}

		content = append(content, bytes.Bytes...)

		// If we got less than the limit, we're done
		if len(bytes.Bytes) < limit {
			break
		}

		offset += int64(len(bytes.Bytes))
	}

	return string(content), nil
}
