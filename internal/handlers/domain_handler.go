package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DNSRecord represents a DNS record
type DNSRecord struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

// Domain represents a domain configuration
type Domain struct {
	ID                string      `json:"id"`
	Domain            string      `json:"domain"`
	Status            string      `json:"status"`
	SSLEnabled        bool        `json:"sslEnabled"`
	CertificateExpiry string      `json:"certificateExpiry"`
	Provider          string      `json:"provider"`
	Records           []DNSRecord `json:"records"`
	CreatedAt         time.Time   `json:"createdAt"`
	UpdatedAt         time.Time   `json:"updatedAt"`
}

// DomainHandler handles domain-related requests
type DomainHandler struct {
	domains []Domain
}

// NewDomainHandler creates a new domain handler
func NewDomainHandler() *DomainHandler {
	now := time.Now()

	// Mock data for demonstration
	domains := []Domain{
		{
			ID:                "domain-1",
			Domain:            "api.aigateway.com",
			Status:            "active",
			SSLEnabled:        true,
			CertificateExpiry: "2024-12-31",
			Provider:          "Cloudflare",
			Records: []DNSRecord{
				{Type: "A", Name: "@", Value: "192.168.1.100", TTL: 300},
				{Type: "CNAME", Name: "www", Value: "api.aigateway.com", TTL: 300},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:                "domain-2",
			Domain:            "dashboard.aigateway.com",
			Status:            "active",
			SSLEnabled:        true,
			CertificateExpiry: "2024-11-15",
			Provider:          "AWS Route 53",
			Records: []DNSRecord{
				{Type: "A", Name: "@", Value: "192.168.1.101", TTL: 300},
				{Type: "MX", Name: "@", Value: "mail.aigateway.com", TTL: 300},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:                "domain-3",
			Domain:            "staging.aigateway.com",
			Status:            "pending",
			SSLEnabled:        false,
			CertificateExpiry: "",
			Provider:          "Google Cloud DNS",
			Records: []DNSRecord{
				{Type: "A", Name: "@", Value: "192.168.1.102", TTL: 300},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	return &DomainHandler{
		domains: domains,
	}
}

// GetDomains returns all domains
func (h *DomainHandler) GetDomains(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    h.domains,
	})
}

// CreateDomain creates a new domain
func (h *DomainHandler) CreateDomain(c *gin.Context) {
	var req Domain
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

	h.domains = append(h.domains, req)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    req,
	})
}

// UpdateDomain updates an existing domain
func (h *DomainHandler) UpdateDomain(c *gin.Context) {
	id := c.Param("id")
	var req Domain
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

	for i, domain := range h.domains {
		if domain.ID == id {
			req.ID = id
			req.CreatedAt = domain.CreatedAt
			req.UpdatedAt = time.Now()
			h.domains[i] = req

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
			"message": "Domain not found",
		},
	})
}

// DeleteDomain deletes a domain
func (h *DomainHandler) DeleteDomain(c *gin.Context) {
	id := c.Param("id")

	for i, domain := range h.domains {
		if domain.ID == id {
			h.domains = append(h.domains[:i], h.domains[i+1:]...)
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Domain deleted successfully",
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Domain not found",
		},
	})
}

// ToggleDomainSSL toggles SSL for a domain
func (h *DomainHandler) ToggleDomainSSL(c *gin.Context) {
	id := c.Param("id")

	for i, domain := range h.domains {
		if domain.ID == id {
			h.domains[i].SSLEnabled = !domain.SSLEnabled
			h.domains[i].UpdatedAt = time.Now()

			// Update certificate expiry based on SSL status
			if h.domains[i].SSLEnabled {
				h.domains[i].CertificateExpiry = time.Now().AddDate(0, 3, 0).Format("2006-01-02")
			} else {
				h.domains[i].CertificateExpiry = ""
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    h.domains[i],
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Domain not found",
		},
	})
}

// RenewDomainCertificate renews the certificate for a domain
func (h *DomainHandler) RenewDomainCertificate(c *gin.Context) {
	id := c.Param("id")

	for i, domain := range h.domains {
		if domain.ID == id {
			if !domain.SSLEnabled {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "SSL_DISABLED",
						"message": "SSL is not enabled for this domain",
					},
				})
				return
			}

			// Extend certificate expiry by 90 days
			h.domains[i].CertificateExpiry = time.Now().AddDate(0, 0, 90).Format("2006-01-02")
			h.domains[i].UpdatedAt = time.Now()

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    h.domains[i],
				"message": "Certificate renewed successfully",
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Domain not found",
		},
	})
}

// RegisterDomainRoutes registers all domain-related routes
func RegisterDomainRoutes(r *gin.Engine, handler *DomainHandler) {
	api := r.Group("/api/v1")

	// Domains
	api.GET("/domains", handler.GetDomains)
	api.POST("/domains", handler.CreateDomain)
	api.PUT("/domains/:id", handler.UpdateDomain)
	api.DELETE("/domains/:id", handler.DeleteDomain)
	api.POST("/domains/:id/ssl", handler.ToggleDomainSSL)
	api.POST("/domains/:id/renew-certificate", handler.RenewDomainCertificate)
}
