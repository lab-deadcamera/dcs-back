package provider

import (
	"dcs-back-v0/internal/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ─── Provider Handlers ──────────────────────────────────────────

func (h *Handler) CreateProvider(c *gin.Context) {
	var req CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	p, err := h.svc.CreateProvider(req)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Created(c, p)
}

func (h *Handler) GetProvider(c *gin.Context) {
	id := c.Param("id")
	p, err := h.svc.GetProviderWithModels(id)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if p == nil {
		utils.NotFound(c, "provider not found")
		return
	}
	utils.Success(c, p)
}

func (h *Handler) ListProviders(c *gin.Context) {
	providers, err := h.svc.ListProvidersWithModels()
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if providers == nil {
		providers = []ProviderWithModels{}
	}
	utils.Success(c, providers)
}

func (h *Handler) UpdateProvider(c *gin.Context) {
	id := c.Param("id")
	var req UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	p, err := h.svc.UpdateProvider(id, req)
	if err != nil {
		if err.Error() == "provider not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, p)
}

func (h *Handler) SoftDeleteProvider(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.SoftDeleteProvider(id); err != nil {
		if err.Error() == "provider not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "provider deleted")
}

// ─── Model Handlers ─────────────────────────────────────────────

func (h *Handler) CreateModel(c *gin.Context) {
	var req CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	m, err := h.svc.CreateModel(req)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Created(c, m)
}

func (h *Handler) GetModel(c *gin.Context) {
	id := c.Param("id")
	m, err := h.svc.GetModelByID(id)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if m == nil {
		utils.NotFound(c, "model not found")
		return
	}
	utils.Success(c, m)
}

func (h *Handler) ListModels(c *gin.Context) {
	models, err := h.svc.ListModels()
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	if models == nil {
		models = []ModelWithProvider{}
	}
	utils.Success(c, models)
}

func (h *Handler) ListModelsByProvider(c *gin.Context) {
	providerID := c.Param("id")
	models, err := h.svc.ListModelsByProvider(providerID)
	if err != nil {
		if err.Error() == "provider not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	if models == nil {
		models = []Model{}
	}
	utils.Success(c, models)
}

func (h *Handler) UpdateModel(c *gin.Context) {
	id := c.Param("id")
	var req UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	m, err := h.svc.UpdateModel(id, req)
	if err != nil {
		if err.Error() == "model not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, m)
}

func (h *Handler) SoftDeleteModel(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.SoftDeleteModel(id); err != nil {
		if err.Error() == "model not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Message(c, "model deleted")
}

func (h *Handler) SetFavorite(c *gin.Context) {
	id := c.Param("id")
	m, err := h.svc.SetFavorite(id)
	if err != nil {
		if err.Error() == "model not found" {
			utils.NotFound(c, err.Error())
			return
		}
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, m)
}
