package embedders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/akhmy/vecrec-core/internal/service"
	"github.com/go-playground/validator/v10"
)

// Учётные данные, необходимые внутреннему провайдеру
type localCredentials struct {
	URL string
}

// Функция, парсящая учетные данные внутреннего провайдера из словаря
func parseLocalCredentials(credentials map[string]any) (*localCredentials, error) {
	url, ok := credentials["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("failed to fetch 'url'")
	}

	return &localCredentials{URL: url}, nil
}

type localEmbedder struct {
	client   *http.Client
	validate *validator.Validate
}

//nolint:ireturn // DIP constructor
func NewLocalEmbedder(client *http.Client, validate *validator.Validate) service.Embedder {
	return &localEmbedder{client: client, validate: validate}
}

// Формат запроса ко внутреннему провайдеру
type localEmbedderRequest struct {
	Strings []string `json:"strings"`
}

// Формат ответа от внутреннего провайдера
type localEmbedderResponse struct {
	Embeddings [][]float32 `json:"embeddings" validate:"required"`
}

// Функция, отправляющая запрос внутреннему провайдеру
func (e localEmbedder) Embed(ctx context.Context, batch []string, credentials map[string]any) ([][]float32, error) {
	lc, err := parseLocalCredentials(credentials)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(localEmbedderRequest{Strings: batch})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := e.client.Post(lc.URL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to r/w request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &UnexpectedResponseStatusError{Has: resp.StatusCode, Want: http.StatusOK}
	}

	var response localEmbedderResponse

	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if err = e.validate.Struct(&response); err != nil {
		return nil, fmt.Errorf("failed to validate response: %w", err)
	}

	return response.Embeddings, nil
}
