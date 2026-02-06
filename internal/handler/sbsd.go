package handler

import (
	"encoding/json"
	"net/http"

	"gerador_cookies/internal/config"
	"gerador_cookies/internal/errors"
	"gerador_cookies/internal/response"
	"gerador_cookies/internal/service"
)

type SbsdRequest struct {
	URL            string `json:"url"`
	AkamaiURL      string `json:"akamaiUrl"`
	Proxy          string `json:"proxy"`
	RandomUA       string `json:"randomUserAgent"`
	UserAgent      string `json:"userAgent"`
	SecChUa        string `json:"secChUa"`
	Language       string `json:"language"`
	AkamaiProvider string `json:"akamaiProvider"`
	GenerateReport bool   `json:"generateReport"`
}

type SbsdHandler struct {
	config  *config.Config
	service *service.SolverService
}

func NewSbsdHandler(cfg *config.Config, svc *service.SolverService) *SbsdHandler {
	return &SbsdHandler{
		config:  cfg,
		service: svc,
	}
}

func (h *SbsdHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// 1. Decode request
	var req SbsdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, &response.ErrorResponse{
			Success: false,
			Error: &response.ErrorDetail{
				Step:        "request_decode",
				StepNumber:  0,
				Description: "Falha ao decodificar request JSON",
				RawError:    err.Error(),
				Retryable:   false,
			},
		})
		return
	}

	// 2. Validar campos obrigatórios
	if req.URL == "" {
		response.WriteError(w, http.StatusBadRequest, &response.ErrorResponse{
			Success: false,
			Error: &response.ErrorDetail{
				Step:        "request_validation",
				StepNumber:  0,
				Description: "Campo 'url' é obrigatório",
				RawError:    "missing required field: url",
				Retryable:   false,
			},
		})
		return
	}

	// 3. Aplicar defaults
	h.applyDefaults(&req)

	// 4. Executar fluxo SbSd
	result, err := h.service.GenerateSbsd(r.Context(), &service.SbsdInput{
		Domain:         req.URL,
		AkamaiURL:      req.AkamaiURL,
		Proxy:          req.Proxy,
		ProfileType:    req.RandomUA,
		UserAgent:      req.UserAgent,
		SecChUa:        req.SecChUa,
		Language:       req.Language,
		AkamaiProvider: req.AkamaiProvider,
		GenerateReport: req.GenerateReport,
	})

	// 5. Tratar erro
	if err != nil {
		if solverErr, ok := err.(*errors.SolverError); ok {
			errResp := solverErr.ToErrorResponse()

			if result != nil && result.PartialCookies != nil {
				errors.WithPartialCookies(errResp, result.PartialCookies)
			}
			if req.GenerateReport && result != nil && result.ReportPath != "" {
				errors.WithDebug(errResp, result.ReportPath)
			}

			response.WriteError(w, solverErr.HTTPStatus(), errResp)
			return
		}

		// Erro genérico
		response.WriteError(w, http.StatusInternalServerError, &response.ErrorResponse{
			Success: false,
			Error: &response.ErrorDetail{
				Step:        "unknown",
				Description: "Erro interno do servidor",
				RawError:    err.Error(),
				Retryable:   false,
			},
		})
		return
	}

	// 6. Sucesso
	response.WriteSuccess(w, &response.SuccessResponse{
		Success:   true,
		Cookies:   result.Cookies,
		Telemetry: result.Telemetry,
		Session:   result.Session,
	})
}

func (h *SbsdHandler) applyDefaults(req *SbsdRequest) {
	if req.RandomUA == "" {
		req.RandomUA = "chrome_144"
	}
	if req.UserAgent == "" {
		req.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"
	}
	if req.SecChUa == "" {
		req.SecChUa = `"Not(A:Brand";v="8", "Chromium";v="144", "Google Chrome";v="144"`
	}
	if req.Language == "" {
		req.Language = "en-US"
	}
	if req.AkamaiProvider == "" {
		req.AkamaiProvider = "jevi"
	}
}
