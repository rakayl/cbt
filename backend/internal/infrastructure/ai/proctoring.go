package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ProctoringClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewProctoringClient(baseURL string) *ProctoringClient {
	return &ProctoringClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ─── REQUEST / RESPONSE TYPES ────────────────────────────────────────────────

type DetectRequest struct {
	ImageBase64 string `json:"image_base64"`
	AttemptID   string `json:"attempt_id"`
}

type FaceDetectResult struct {
	FaceCount   int     `json:"face_count"`
	HasFace     bool    `json:"has_face"`
	Confidence  float64 `json:"confidence"`
	BoundingBox []int   `json:"bounding_box,omitempty"`
}

type VerifyRequest struct {
	ImageBase64  string  `json:"image_base64"`
	BaseEmbedding []float64 `json:"base_embedding"`
	AttemptID    string  `json:"attempt_id"`
}

type VerifyResult struct {
	Match      bool    `json:"match"`
	Similarity float64 `json:"similarity"`
	Threshold  float64 `json:"threshold"`
}

type EmbeddingRequest struct {
	ImageBase64 string `json:"image_base64"`
}

type EmbeddingResult struct {
	Embedding []float64 `json:"embedding"`
	Success   bool      `json:"success"`
}

// ─── METHODS ─────────────────────────────────────────────────────────────────

// DetectFace calls Python service to detect faces in image
func (c *ProctoringClient) DetectFace(ctx context.Context, req DetectRequest) (*FaceDetectResult, error) {
	var result FaceDetectResult
	if err := c.post(ctx, "/detect/face", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// VerifyFace compares current frame against baseline embedding
func (c *ProctoringClient) VerifyFace(ctx context.Context, req VerifyRequest) (*VerifyResult, error) {
	var result VerifyResult
	if err := c.post(ctx, "/verify/face", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GenerateEmbedding creates face embedding from image for baseline
func (c *ProctoringClient) GenerateEmbedding(ctx context.Context, req EmbeddingRequest) (*EmbeddingResult, error) {
	var result EmbeddingResult
	if err := c.post(ctx, "/embedding/generate", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DetectObjects calls YOLO endpoint for phone/object detection
func (c *ProctoringClient) DetectObjects(ctx context.Context, req DetectRequest) ([]string, error) {
	var result struct {
		Objects []string `json:"objects"`
	}
	if err := c.post(ctx, "/detect/objects", req, &result); err != nil {
		return nil, err
	}
	return result.Objects, nil
}

// ─── HELPER ───────────────────────────────────────────────────────────────────

func (c *ProctoringClient) post(ctx context.Context, path string, body interface{}, out interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("proctoring service error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("proctoring service returned status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
