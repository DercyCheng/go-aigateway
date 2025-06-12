package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Certificate represents a SSL/TLS certificate
type Certificate struct {
	ID              string    `json:"id"`
	Domain          string    `json:"domain"`
	Provider        string    `json:"provider"`
	Status          string    `json:"status"`
	ExpiryDate      string    `json:"expiryDate"`
	AutoRenew       bool      `json:"autoRenew"`
	LastRenewed     string    `json:"lastRenewed"`
	CertificateType string    `json:"certificateType"`
	Algorithm       string    `json:"algorithm"`
	KeySize         int       `json:"keySize"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// CertificateHandler handles certificate-related requests
type CertificateHandler struct {
	certificates []Certificate
}

// NewCertificateHandler creates a new certificate handler
func NewCertificateHandler() *CertificateHandler {
	now := time.Now()

	// Mock data for demonstration
	certificates := []Certificate{
		{
			ID:              "cert-1",
			Domain:          "api.aigateway.com",
			Provider:        "Let's Encrypt",
			Status:          "active",
			ExpiryDate:      "2024-12-31",
			AutoRenew:       true,
			LastRenewed:     "2024-06-01",
			CertificateType: "Domain Validated",
			Algorithm:       "RSA",
			KeySize:         2048,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "cert-2",
			Domain:          "*.aigateway.com",
			Provider:        "Cloudflare",
			Status:          "active",
			ExpiryDate:      "2024-11-15",
			AutoRenew:       true,
			LastRenewed:     "2024-05-15",
			CertificateType: "Wildcard",
			Algorithm:       "ECDSA",
			KeySize:         256,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "cert-3",
			Domain:          "dashboard.aigateway.com",
			Provider:        "DigiCert",
			Status:          "expiring",
			ExpiryDate:      "2024-07-01",
			AutoRenew:       false,
			LastRenewed:     "2023-07-01",
			CertificateType: "Extended Validation",
			Algorithm:       "RSA",
			KeySize:         4096,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}

	return &CertificateHandler{
		certificates: certificates,
	}
}

// GetCertificates returns all certificates
func (h *CertificateHandler) GetCertificates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    h.certificates,
	})
}

// CreateCertificate creates a new certificate
func (h *CertificateHandler) CreateCertificate(c *gin.Context) {
	var req Certificate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
		return
	}

	now := time.Now()
	req.ID = generateID()
	req.CreatedAt = now
	req.UpdatedAt = now
	req.Status = "pending"

	h.certificates = append(h.certificates, req)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    req,
	})
}

// UpdateCertificate updates an existing certificate
func (h *CertificateHandler) UpdateCertificate(c *gin.Context) {
	id := c.Param("id")
	var req Certificate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
		return
	}

	for i, certificate := range h.certificates {
		if certificate.ID == id {
			req.ID = id
			req.CreatedAt = certificate.CreatedAt
			req.UpdatedAt = time.Now()
			h.certificates[i] = req

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    req,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Certificate not found",
		},
	})
}

// DeleteCertificate deletes a certificate
func (h *CertificateHandler) DeleteCertificate(c *gin.Context) {
	id := c.Param("id")

	for i, certificate := range h.certificates {
		if certificate.ID == id {
			h.certificates = append(h.certificates[:i], h.certificates[i+1:]...)
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Certificate deleted successfully",
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Certificate not found",
		},
	})
}

// RenewCertificate renews a certificate
func (h *CertificateHandler) RenewCertificate(c *gin.Context) {
	id := c.Param("id")

	for i, certificate := range h.certificates {
		if certificate.ID == id {
			h.certificates[i].LastRenewed = time.Now().Format("2006-01-02")
			h.certificates[i].Status = "active"
			h.certificates[i].UpdatedAt = time.Now()

			// Extend expiry date by 90 days
			if expiryTime, err := time.Parse("2006-01-02", certificate.ExpiryDate); err == nil {
				h.certificates[i].ExpiryDate = expiryTime.AddDate(0, 0, 90).Format("2006-01-02")
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    h.certificates[i],
				"message": "Certificate renewed successfully",
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Certificate not found",
		},
	})
}

// ToggleCertificateAutoRenew toggles the auto-renew setting of a certificate
func (h *CertificateHandler) ToggleCertificateAutoRenew(c *gin.Context) {
	id := c.Param("id")

	for i, certificate := range h.certificates {
		if certificate.ID == id {
			h.certificates[i].AutoRenew = !certificate.AutoRenew
			h.certificates[i].UpdatedAt = time.Now()

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    h.certificates[i],
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Certificate not found",
		},
	})
}

// RegisterCertificateRoutes registers all certificate-related routes
func RegisterCertificateRoutes(r *gin.Engine, handler *CertificateHandler) {
	api := r.Group("/api/v1")

	// Certificates
	api.GET("/certificates", handler.GetCertificates)
	api.POST("/certificates", handler.CreateCertificate)
	api.PUT("/certificates/:id", handler.UpdateCertificate)
	api.DELETE("/certificates/:id", handler.DeleteCertificate)
	api.POST("/certificates/:id/renew", handler.RenewCertificate)
	api.POST("/certificates/:id/auto-renew", handler.ToggleCertificateAutoRenew)
}
