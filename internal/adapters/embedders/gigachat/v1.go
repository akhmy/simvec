package gigachat

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/akhmy/vecrec-core/internal/service"
	"github.com/go-playground/validator/v10"
)

// Учетные данные, необходимые GigaChat API
type gigaChatV1Credentials struct {
	EmbedURL              string
	RefreshAccessTokenURL string
	AuthToken             string
	Model                 string
	Scope                 string
}

// Функция, парсящая учетные данные провайдера
func parseGigaChatV1Credentials(credentials map[string]any) (*gigaChatV1Credentials, error) {
	getString := func(key string) (string, error) {
		v, ok := credentials[key].(string)
		if !ok || v == "" {
			return "", fmt.Errorf("missing or invalid credential field %q", key)
		}
		return v, nil
	}

	embedURL, err := getString("embed_url")
	if err != nil {
		return nil, err
	}
	refreshURL, err := getString("refresh_access_token_url")
	if err != nil {
		return nil, err
	}
	authToken, err := getString("auth_token")
	if err != nil {
		return nil, err
	}
	model, err := getString("model")
	if err != nil {
		return nil, err
	}
	scope, err := getString("scope")
	if err != nil {
		return nil, err
	}

	return &gigaChatV1Credentials{
		EmbedURL:              embedURL,
		RefreshAccessTokenURL: refreshURL,
		AuthToken:             authToken,
		Model:                 model,
		Scope:                 scope,
	}, nil
}

// Формат записи в кэше токенов
type tokenEntry struct {
	accessToken string
	expiresAt   time.Time
}

type gigaChatV1Embedder struct {
	client   *http.Client
	validate *validator.Validate

	mu    sync.Mutex
	cache map[string]tokenEntry // кэш токенов
}

func NewGigaChatV1Embedder(client *http.Client, validate *validator.Validate) service.Embedder {
	return &gigaChatV1Embedder{
		client:   client,
		validate: validate,
		cache:    make(map[string]tokenEntry),
	}
}

type gigaChatV1EmbedderRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type gigaChatV1EmbedderResponse struct {
	Data []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// Максимальный размер пакета строк, который можно отправить провайдеру.
// Подобран экспериментально, документация не содержит точного значения.
const maxBatchSize = 100

func (e *gigaChatV1Embedder) Embed(
	ctx context.Context, batch []string, rawCredentials map[string]any,
) ([][]float32, error) {
	credentials, err := parseGigaChatV1Credentials(rawCredentials)
	if err != nil {
		return nil, err
	}

	result := make([][]float32, 0, len(batch))
	for i := 0; i < len(batch); i += maxBatchSize {
		end := min(i+maxBatchSize, len(batch))

		accessToken, err := e.getAccessToken(ctx, credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to get access token: %w", err)
		}

		payload, err := json.Marshal(gigaChatV1EmbedderRequest{Model: credentials.Model, Input: batch[i:end]})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, credentials.EmbedURL, bytes.NewBuffer(payload))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := e.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to r/w request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("chunk [%d:%d] unexpected status: %d, body: %s", i, end, resp.StatusCode, body)
		}

		var embedResp gigaChatV1EmbedderResponse
		if err = json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		chunk := make([][]float32, len(embedResp.Data))
		for _, item := range embedResp.Data {
			chunk[item.Index] = item.Embedding
		}
		result = append(result, chunk...)
	}

	return result, nil
}

// Функция, проверяющая, есть ли токен доступа и действителен ли он. Если нет -- запрашивает новый.
func (e *gigaChatV1Embedder) getAccessToken(ctx context.Context, credentials *gigaChatV1Credentials) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	entry, ok := e.cache[credentials.AuthToken]
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.accessToken, nil
	}

	accessToken, expiresAt, err := e.requestAccessToken(ctx, credentials)
	if err != nil {
		return "", fmt.Errorf("failed to request new access token: %w", err)
	}

	e.cache[credentials.AuthToken] = tokenEntry{accessToken: accessToken, expiresAt: expiresAt.Add(-30 * time.Second)}

	return accessToken, nil
}

type oauthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

func (e *gigaChatV1Embedder) requestAccessToken(
	ctx context.Context, credentials *gigaChatV1Credentials,
) (string, time.Time, error) {
	body := url.Values{}
	body.Set("scope", credentials.Scope)

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, credentials.RefreshAccessTokenURL,
		strings.NewReader(body.Encode()),
	)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}

	rqUID, err := newUUID()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate RqUID: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("RqUID", rqUID)
	req.Header.Set("Authorization", "Basic "+credentials.AuthToken)

	resp, err := e.client.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", time.Time{}, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
	}

	var oauthResp oauthResponse

	err = json.NewDecoder(resp.Body).Decode(&oauthResp)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to decode response: %w", err)
	}

	expiresAt := time.UnixMilli(oauthResp.ExpiresAt)

	return oauthResp.AccessToken, expiresAt, nil
}

// Функция, генерирующая UUID для запросов токенов доступа
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
